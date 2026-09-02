package main

import (
 "encoding/json"
 "fmt"
 "os"
 "path/filepath"
 "strings"
 "github.com/anonyxhappie/keystone/internal/context"
 "github.com/anonyxhappie/keystone/internal/domain"
 "github.com/anonyxhappie/keystone/internal/harness"
 "github.com/anonyxhappie/keystone/internal/project"
 "github.com/anonyxhappie/keystone/internal/runtime"
 "github.com/anonyxhappie/keystone/internal/state"
 "github.com/anonyxhappie/keystone/internal/validation"
 "github.com/anonyxhappie/keystone/internal/work"
)
func main(){if len(os.Args)<2{usage();return};root,_:=os.Getwd();switch os.Args[1]{case "init":runInit(root);case "status":runStatus(root);case "ask":runAsk(root,os.Args[2:]);case "continue":runContinue(root);case "validate":runValidate(root);case "state":runState(os.Args[2:]);case "doctor":fmt.Println("Keystone doctor: OK");default:usage();os.Exit(2)}}
func usage(){fmt.Println("keystone init | status | ask <request> | continue | validate | state demo | doctor")}
func runInit(root string){name:=filepath.Base(root);caps:=project.Detect(root);p,err:=state.New(root).Init(name,caps);if err!=nil{fatal(err)};b,_:=json.MarshalIndent(p,"","  ");fmt.Println(string(b))}
func runStatus(root string){var p map[string]any;if err:=state.New(root).Read("project.json",&p);err!=nil{fatal(fmt.Errorf("Keystone is not initialized: %w",err))};b,_:=json.MarshalIndent(p,"","  ");fmt.Println(string(b))}
func runAsk(root string,args []string){if len(args)==0{fatal(fmt.Errorf("ask requires a request"))};req:=strings.Join(args," ");o:=work.NewOrder(req);p:=context.Compile(root,work.Packet(o));if err:=state.New(root).Write("work/"+o.ID+".json",o);err!=nil{fatal(err)};fmt.Println(harness.Render(p))}
func runContinue(root string){entries,err:=os.ReadDir(filepath.Join(root,state.Dir,"work"));if err!=nil{fatal(fmt.Errorf("no resumable work: %w",err))};if len(entries)==0{fatal(fmt.Errorf("no work orders found"))};fmt.Printf("Found %d durable work order(s); use the configured harness adapter to resume.\n",len(entries))}
func runValidate(root string){var p domain.Project;if err:=state.New(root).Read("project.json",&p);err!=nil{fatal(err)};plan:=validation.PlanFor(domain.Risk{Level:"low"},p.Capabilities);fmt.Printf("Validation tier %d: %s\n",plan.Tier,strings.Join(plan.Checks,", "))}
func runState(args []string){if len(args)==0||args[0]!="demo"{fmt.Println("usage: keystone state demo");return};m:=runtime.New();path:=[]runtime.State{runtime.Understand,runtime.Assess,runtime.Plan,runtime.Context,runtime.Dispatch,runtime.Execute,runtime.Observe,runtime.Verify,runtime.Evaluate,runtime.Supervise,runtime.Decide};for _,s:=range path{if err:=m.TransitionTo(s,"state-machine progression");err!=nil{fatal(err)}};if err:=m.ApplyDecision(runtime.CompleteDecision,"all checks verified");err!=nil{fatal(err)};b,_:=json.MarshalIndent(m,"","  ");fmt.Println(string(b))}
func fatal(err error){fmt.Fprintln(os.Stderr,"keystone:",err);os.Exit(1)}
