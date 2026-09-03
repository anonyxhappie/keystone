package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/anonyxhappie/keystone/internal/domain"
)

// Compile preserves the V1 API while selecting a small, provenance-bearing set
// of repository files. It never reads file contents into the work packet.
func Compile(root string, packet domain.WorkPacket) domain.WorkPacket {
	return CompileWithImpact(root, packet, nil)
}

// CompileWithImpact compiles repository context without budget constraints.
func CompileWithImpact(root string, packet domain.WorkPacket, changed []string) domain.WorkPacket {
	p, _ := PlanContext(root, packet, changed, 0)
	return p
}

// PlanContext compiles and intelligently re-plans context to fit within the given budget.
// If budget <= 0, no budget constraint is enforced.
func PlanContext(root string, packet domain.WorkPacket, changed []string, budget int) (domain.WorkPacket, error) {
	packet.SchemaVersion = "2"
	selected := map[string]domain.ContextRef{}
	add := func(path, typ, reason, source string, relevance float64) {
		path = filepath.ToSlash(filepath.Clean(path))
		if path == "." || strings.HasPrefix(path, "../") {
			return
		}
		if old, ok := selected[path]; ok {
			if relevance > old.Relevance {
				old.Relevance = relevance
				selected[path] = old
			}
			return
		}
		selected[path] = domain.ContextRef{Type: typ, Path: path, Reason: reason, Source: source, Relevance: relevance}
	}

	for _, file := range changed {
		add(file, "changed-file", "directly changed by current work", "git", 1.0)
	}

	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == ".keystone" || entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		name := entry.Name()
		low := strings.ToLower(name)
		if name == "README.md" || name == "AGENTS.md" || name == "CLAUDE.md" || name == ".cursorrules" {
			relevance := 0.95
			if filepath.Dir(rel) != "." {
				relevance = 0.70
			}
			add(rel, "instruction", "project instructions and constraints", "repository", relevance)
		}
		if strings.Contains(low, "architecture") || strings.Contains(low, "decision") {
			add(rel, "architecture", "architecture or recorded decision", "repository", 0.80)
		}
		if strings.HasSuffix(low, "_test.go") || strings.HasSuffix(low, ".test.ts") || strings.HasSuffix(low, ".test.js") || strings.HasPrefix(low, "test_") {
			add(rel, "test", "potentially relevant validation", "repository", 0.55)
		}
		return nil
	})

	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "that": true, "this": true,
		"from": true, "into": true, "what": true, "does": true, "make": true, "current": true,
		"project": true, "repo": true, "repository": true, "right": true, "here": true,
		"audit": false, // informative keyword
	}

	rawWords := strings.Fields(strings.ToLower(packet.Objective))
	keywords := []string{}
	for _, w := range rawWords {
		w = strings.Trim(w, `.,:;!?'"()[]{}`)
		if len(w) > 3 && !stopWords[w] {
			keywords = append(keywords, w)
		}
	}

	files := make([]string, 0, len(selected))
	for path := range selected {
		files = append(files, path)
	}
	sort.Strings(files)

	for _, path := range files {
		base := strings.ToLower(filepath.Base(path))
		pathLower := strings.ToLower(path)
		matches := 0
		for _, kw := range keywords {
			if strings.Contains(base, kw) || strings.Contains(pathLower, kw) {
				matches++
			}
		}
		if matches > 0 {
			ref := selected[path]
			boost := 0.05 * float64(matches)
			newRel := ref.Relevance + boost
			if newRel < 0.75 {
				newRel = 0.75
			}
			if newRel > 0.90 && ref.Type != "changed-file" && ref.Type != "instruction" {
				newRel = 0.90
			}
			ref.Relevance = newRel
			if ref.Reason == "" || ref.Reason == "potentially relevant validation" {
				ref.Reason = "objective keyword matches file name"
			}
			selected[path] = ref
		}
	}

	refs := make([]domain.ContextRef, 0, len(selected))
	for _, path := range files {
		ref := selected[path]
		ref.TokenEstimate = estimate(root, ref.Path)
		refs = append(refs, ref)
	}

	// Sort candidates deterministically: relevance descending, then path ascending
	sortRefs(refs)

	if budget <= 0 {
		if len(refs) > 64 {
			refs = refs[:64]
		}
		decisions := make([]domain.ContextDecision, len(refs))
		totalTokens := 0
		for i, r := range refs {
			totalTokens += r.TokenEstimate
			decisions[i] = domain.ContextDecision{
				Path:          r.Path,
				Type:          r.Type,
				Action:        "retained",
				Reason:        "retained (no budget constraint)",
				Relevance:     r.Relevance,
				TokenEstimate: r.TokenEstimate,
			}
		}
		packet.Context = refs
		packet.ContextDecisions = decisions
		packet.ContextTokens = totalTokens
		packet.ContextBudget = 0
		return packet, nil
	}

	// Check if initial compilation already fits within budget
	initialTokens := 0
	for _, r := range refs {
		initialTokens += r.TokenEstimate
	}

	if initialTokens <= budget {
		decisions := make([]domain.ContextDecision, len(refs))
		for i, r := range refs {
			decisions[i] = domain.ContextDecision{
				Path:          r.Path,
				Type:          r.Type,
				Action:        "retained",
				Reason:        "retained (fits within context budget)",
				Relevance:     r.Relevance,
				TokenEstimate: r.TokenEstimate,
			}
		}
		packet.Context = refs
		packet.ContextDecisions = decisions
		packet.ContextTokens = initialTokens
		packet.ContextBudget = budget
		return packet, nil
	}

	// Context exceeds budget: initiate multi-stage intelligent re-planning
	// 1. Enforce mandatory items guardrail
	mandatoryTokens := 0
	mandatoryPaths := []string{}
	for _, r := range refs {
		if isMandatory(r) {
			mandatoryTokens += r.TokenEstimate
			mandatoryPaths = append(mandatoryPaths, r.Path)
		}
	}

	if mandatoryTokens > budget {
		return packet, fmt.Errorf("context budget exceeded: mandatory context items (%d tokens: %s) exceed configured budget (%d tokens); cannot safely proceed without truncating mandatory instructions or evidence", mandatoryTokens, strings.Join(mandatoryPaths, ", "), budget)
	}

	// 2. Pass 1: Subsystem clustering & redundancy reduction
	// Group non-mandatory items by directory
	subsystems := map[string][]int{}
	for i, r := range refs {
		if !isMandatory(r) {
			dir := filepath.Dir(r.Path)
			subsystems[dir] = append(subsystems[dir], i)
		}
	}

	compressIndices := map[int]bool{}
	for _, indices := range subsystems {
		if len(indices) > 3 {
			// Keep top 3 by relevance as candidates for raw inclusion,
			// mark the remaining indices in the subsystem for structural compression
			for rank, idx := range indices {
				if rank >= 3 {
					compressIndices[idx] = true
				}
			}
		}
	}

	// 3. Pass 2: Hierarchical Structural Compression
	// Compress marked items, tests, secondary docs, or large files (> 400 tokens)
	for i := range refs {
		if isMandatory(refs[i]) {
			continue
		}
		if compressIndices[i] || refs[i].Type == "test" || (refs[i].Type == "instruction" && filepath.Dir(refs[i].Path) != ".") || refs[i].TokenEstimate > 400 {
			summary, summaryTokens := CompressFile(root, refs[i].Path)
			refs[i].Compressed = true
			refs[i].OriginalTokens = refs[i].TokenEstimate
			refs[i].TokenEstimate = summaryTokens
			refs[i].Summary = summary
		}
	}

	compressedTokens := 0
	for _, r := range refs {
		compressedTokens += r.TokenEstimate
	}

	if compressedTokens <= budget {
		decisions := make([]domain.ContextDecision, len(refs))
		for i, r := range refs {
			if r.Compressed {
				decisions[i] = domain.ContextDecision{
					Path:           r.Path,
					Type:           r.Type,
					Action:         "compressed",
					Reason:         "compressed to structural outline to fit context budget",
					Relevance:      r.Relevance,
					OriginalTokens: r.OriginalTokens,
					TokenEstimate:  r.TokenEstimate,
				}
			} else {
				decisions[i] = domain.ContextDecision{
					Path:          r.Path,
					Type:          r.Type,
					Action:        "retained",
					Reason:        "retained in full within context budget",
					Relevance:     r.Relevance,
					TokenEstimate: r.TokenEstimate,
				}
			}
		}
		packet.Context = refs
		packet.ContextDecisions = decisions
		packet.ContextTokens = compressedTokens
		packet.ContextBudget = budget
		return packet, nil
	}

	// 4. Pass 3: Greedy Priority & Relevance-Ranked Packing
	// Sort so mandatory items come first, then high-relevance architecture, domain files, then compressed outlines
	sort.SliceStable(refs, func(i, j int) bool {
		mI := isMandatory(refs[i])
		mJ := isMandatory(refs[j])
		if mI != mJ {
			return mI // mandatory items first
		}
		if refs[i].Relevance != refs[j].Relevance {
			return refs[i].Relevance > refs[j].Relevance
		}
		return refs[i].Path < refs[j].Path
	})

	retained := []domain.ContextRef{}
	decisions := []domain.ContextDecision{}
	currentTokens := 0

	for _, r := range refs {
		if isMandatory(r) {
			retained = append(retained, r)
			currentTokens += r.TokenEstimate
			decisions = append(decisions, domain.ContextDecision{
				Path:          r.Path,
				Type:          r.Type,
				Action:        "retained",
				Reason:        "retained mandatory instruction or evidence",
				Relevance:     r.Relevance,
				TokenEstimate: r.TokenEstimate,
			})
			continue
		}

		if currentTokens+r.TokenEstimate <= budget {
			retained = append(retained, r)
			currentTokens += r.TokenEstimate
			action := "retained"
			reason := "retained by relevance ranking within context budget"
			if r.Compressed {
				action = "compressed"
				reason = "compressed to structural outline within context budget"
			}
			decisions = append(decisions, domain.ContextDecision{
				Path:           r.Path,
				Type:           r.Type,
				Action:         action,
				Reason:         reason,
				Relevance:      r.Relevance,
				OriginalTokens: r.OriginalTokens,
				TokenEstimate:  r.TokenEstimate,
			})
		} else {
			decisions = append(decisions, domain.ContextDecision{
				Path:           r.Path,
				Type:           r.Type,
				Action:         "omitted",
				Reason:         fmt.Sprintf("omitted to satisfy %d-token budget (relevance %.2f, requires %d tokens)", budget, r.Relevance, r.TokenEstimate),
				Relevance:      r.Relevance,
				OriginalTokens: r.OriginalTokens,
				TokenEstimate:  r.TokenEstimate,
			})
		}
	}

	packet.Context = retained
	packet.ContextDecisions = decisions
	packet.ContextTokens = currentTokens
	packet.ContextBudget = budget
	return packet, nil
}

func isMandatory(ref domain.ContextRef) bool {
	if ref.Type == "evidence" {
		return true
	}
	if ref.Type == "instruction" {
		base := filepath.Base(ref.Path)
		if base == "AGENTS.md" || base == "CLAUDE.md" || base == ".cursorrules" || (base == "README.md" && (ref.Path == "README.md" || filepath.Dir(ref.Path) == ".")) {
			return true
		}
	}
	return false
}

func sortRefs(refs []domain.ContextRef) {
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Relevance == refs[j].Relevance {
			return refs[i].Path < refs[j].Path
		}
		return refs[i].Relevance > refs[j].Relevance
	})
}

func estimate(root, path string) int {
	info, err := os.Stat(filepath.Join(root, path))
	if err != nil {
		return 0
	}
	return int(info.Size() / 4)
}
