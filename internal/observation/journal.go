package observation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Event struct {
	ID string `json:"id"`
	RunID string `json:"runId"`
	Sequence uint64 `json:"sequence"`
	Type string `json:"type"`
	Source string `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Payload map[string]any `json:"payload,omitempty"`
}

type Journal struct { path string; mu sync.Mutex; sequence uint64 }

func Open(path string) (*Journal,error) {
	f,err:=os.Open(path); if err!=nil { if os.IsNotExist(err) { return &Journal{path:path},nil }; return nil,err }
	defer f.Close(); var n uint64; s:=bufio.NewScanner(f)
	for s.Scan(){ var e Event; if json.Unmarshal(s.Bytes(),&e)==nil && e.Sequence>n { n=e.Sequence } }
	if err:=s.Err(); err!=nil{return nil,err}; return &Journal{path:path,sequence:n},nil
}

func (j *Journal) Append(e Event) error {
	j.mu.Lock(); defer j.mu.Unlock(); j.sequence++; e.Sequence=j.sequence; if e.ID=="" { e.ID=fmt.Sprintf("OBS-%d",e.Sequence) }; if e.Timestamp.IsZero(){e.Timestamp=time.Now().UTC()}
	if err:=os.MkdirAll(filepathDir(j.path),0755);err!=nil{return err}; f,err:=os.OpenFile(j.path,os.O_CREATE|os.O_WRONLY|os.O_APPEND,0600);if err!=nil{return err};defer f.Close(); b,err:=json.Marshal(e);if err!=nil{return err};_,err=f.Write(append(b,'\n'));return err
}

func filepathDir(p string) string { for i:=len(p)-1;i>=0;i-- { if p[i]=='/' { if i==0{return "/"};return p[:i] } }; return "." }
