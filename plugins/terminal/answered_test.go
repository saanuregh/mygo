package terminal

import "testing"

// A terminal whose Conn's other end answers the program's questions
// writes no answers of its own, but still sends what is typed.
func TestAnsweredByConn(t *testing.T) {
	loadLib(t)
	for _, answered := range []bool{false, true} {
		p := newPipe()
		term, err := New(Options{Conn: p, AnsweredByConn: answered})
		if err != nil {
			t.Fatal(err)
		}
		// A cursor position report, as shells and editors ask.
		term.Feed([]byte("ab\x1b[6n"))
		got := p.take(term)
		want := "\x1b[1;3R"
		if answered {
			want = ""
		}
		if got != want {
			t.Errorf("answered %v: the program got %q, want %q", answered, got, want)
		}
		term.Send([]byte("ls\r"))
		if got := p.take(term); got != "ls\r" {
			t.Errorf("answered %v: typed %q, want %q", answered, got, "ls\r")
		}
		term.Close()
	}
}
