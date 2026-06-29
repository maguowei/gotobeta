package request

import "testing"

func TestCreateTodoRequestToCommand(t *testing.T) {
	req := CreateTodoRequest{Title: "write tests"}

	cmd := req.ToCommand()
	if cmd.Title != "write tests" {
		t.Fatalf("Title = %q, want write tests", cmd.Title)
	}
}
