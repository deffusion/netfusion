package node

import (
	"fmt"
	mapset "github.com/deckarep/golang-set/v2"
	"testing"
)

func TestSet(t *testing.T) {
	s := (mapset.NewSet[string])()
	s.Add("hello")
	fmt.Println(s.Contains("hello"))
	fmt.Println(s.Contains("hello1"))
}
