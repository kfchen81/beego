package vanilla

import (
	"context"
	"encoding/csv"
	"fmt"
	_ "github.com/kfchen81/beego/testing"
	"github.com/pkg/profile"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"testing"
	"time"
)



func TestResource_LoginAsPprof(t *testing.T) {
	go func() {
		http.ListenAndServe("0.0.0.0:8899", nil)
	}()
	defer profile.Start(profile.MemProfile).Stop()
	const N = 100
	seed := time.Now().UTC().UnixNano()
	t.Logf("Random seed: %d", seed)
	rng := rand.New(rand.NewSource(seed))
	pwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	corpFile := path.Join(path.Dir(pwd), "/dev/auth_user.csv")
	corpAccounts := loginAccounts(corpFile)

	ctx := context.Background()
	resource := NewResource(ctx)
	for i := 0; i < N; i++ {
		account := corpAccounts[rng.Intn(len(corpAccounts)-1)]
		resource.LoginAs(account[4])
	}
}

func TestResource_LoginAs(t *testing.T) {
	ctx := context.Background()
	resource := NewResource(ctx)
	resource.LoginAs("manager")
}

func loginAccounts(file string) (accounts [][]string) {
	csvFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()

	csvReader := csv.NewReader(csvFile)
	csvReader.LazyQuotes = true
	rows, err := csvReader.ReadAll() // `rows` is of type [][]string
	if err != nil {
		panic(err)
	}
	return rows
}

func init() {
}
