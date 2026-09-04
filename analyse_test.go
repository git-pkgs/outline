package outline

import "testing"

func declByName(t *testing.T, decls []decl, name string) (int, decl) {
	t.Helper()
	for i, d := range decls {
		if d.Name == name {
			return i, d
		}
	}
	t.Fatalf("decl %q not found in %v", name, decls)
	return -1, decl{}
}

func callByName(t *testing.T, calls []Call, name string) Call {
	t.Helper()
	for _, c := range calls {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("call %q not found in %v", name, calls)
	return Call{}
}

func TestAnalysePythonNesting(t *testing.T) {
	src := []byte(`import os

class Handler:
    def process(self, path):
        def helper():
            return os.getcwd()
        return helper()

def top():
    Handler().process(".")
`)
	a, ok := analyse(src, "h.py")
	if !ok {
		t.Fatal("analyse returned false")
	}
	if a.Lang != "python" {
		t.Fatalf("lang = %q", a.Lang)
	}

	iH, _ := declByName(t, a.Decls, "Handler")
	iP, dP := declByName(t, a.Decls, "process")
	iHelp, dHelp := declByName(t, a.Decls, "helper")
	_, dTop := declByName(t, a.Decls, "top")

	if a.Decls[iH].Parent != -1 {
		t.Errorf("Handler parent = %d, want -1", a.Decls[iH].Parent)
	}
	if dP.Parent != iH {
		t.Errorf("process parent = %d, want %d (Handler)", dP.Parent, iH)
	}
	if dHelp.Parent != iP {
		t.Errorf("helper parent = %d, want %d (process)", dHelp.Parent, iP)
	}
	if dTop.Parent != -1 {
		t.Errorf("top parent = %d, want -1", dTop.Parent)
	}
	if dP.Kind != "func" || a.Decls[iH].Kind != "class" {
		t.Errorf("kinds: Handler=%q process=%q", a.Decls[iH].Kind, dP.Kind)
	}
	if a.Decls[iH].Start >= dP.Start || dP.End > a.Decls[iH].End {
		t.Errorf("process span [%d,%d) not inside Handler [%d,%d)",
			dP.Start, dP.End, a.Decls[iH].Start, a.Decls[iH].End)
	}

	if len(a.Imports) != 1 || a.Imports[0].Module != "os" {
		t.Errorf("imports = %v", a.Imports)
	}

	getcwd := callByName(t, a.Calls, "getcwd")
	if getcwd.Receiver != "os" || getcwd.In != iHelp {
		t.Errorf("getcwd: receiver=%q in=%d, want os/%d", getcwd.Receiver, getcwd.In, iHelp)
	}
	helperCall := callByName(t, a.Calls, "helper")
	if helperCall.Receiver != "" || helperCall.In != iP {
		t.Errorf("helper call: receiver=%q in=%d, want \"\"/%d", helperCall.Receiver, helperCall.In, iP)
	}
}

func TestAnalyseGoCalls(t *testing.T) {
	src := []byte(`package main

import "os/exec"

type Runner struct{}

func (r Runner) Run(name string) error {
	cmd := exec.Command(name)
	return cmd.Run()
}

func main() {
	check(Runner{}.Run("ls"))
}
`)
	a, ok := analyse(src, "main.go")
	if !ok {
		t.Fatal("analyse returned false")
	}

	iRun, dRun := declByName(t, a.Decls, "Run")
	iMain, _ := declByName(t, a.Decls, "main")
	_, dRunner := declByName(t, a.Decls, "Runner")

	if dRun.Parent != -1 || dRunner.Parent != -1 {
		t.Errorf("Go top-level decls should have parent -1: Run=%d Runner=%d", dRun.Parent, dRunner.Parent)
	}
	if dRun.SigEnd >= dRun.End || dRun.SigEnd <= dRun.Start {
		t.Errorf("Run SigEnd=%d not between Start=%d End=%d", dRun.SigEnd, dRun.Start, dRun.End)
	}

	if len(a.Imports) != 1 || a.Imports[0].Module != "os/exec" {
		t.Errorf("imports = %v", a.Imports)
	}

	cmd := callByName(t, a.Calls, "Command")
	if cmd.Receiver != "exec" || cmd.In != iRun {
		t.Errorf("Command: receiver=%q in=%d, want exec/%d", cmd.Receiver, cmd.In, iRun)
	}
	check := callByName(t, a.Calls, "check")
	if check.Receiver != "" || check.In != iMain {
		t.Errorf("check: receiver=%q in=%d, want \"\"/%d", check.Receiver, check.In, iMain)
	}
}

func TestEnclosing(t *testing.T) {
	decls := []decl{
		{Name: "a", Start: 0, End: 100},
		{Name: "b", Start: 10, End: 50},
		{Name: "c", Start: 60, End: 90},
	}
	cases := []struct {
		pos  uint32
		want int
	}{
		{5, 0}, {20, 1}, {55, 0}, {70, 2}, {200, -1},
	}
	for _, tc := range cases {
		if got := enclosing(decls, tc.pos); got != tc.want {
			t.Errorf("enclosing(%d) = %d, want %d", tc.pos, got, tc.want)
		}
	}
}
