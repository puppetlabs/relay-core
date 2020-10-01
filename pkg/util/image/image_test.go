package image

import (
	"testing"

	"github.com/kr/pretty"
)

func TestImageTagRepo(t *testing.T) {
	ref, _ := RepoReference("relaysh/kubernetes-step-kubectl:latest")

	pretty.Println(ref.Context().RepositoryStr())
}
