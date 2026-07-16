package main

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	count := flag.Int("count", 50, "number of ranked priority passes")
	flag.Parse()
	if *count < 1 || *count > 10000 {
		fmt.Fprintln(os.Stderr, "count must be between 1 and 10000")
		os.Exit(2)
	}
	passes := make([]string, *count)
	for i := range passes {
		var random [10]byte
		if _, err := rand.Read(random[:]); err != nil {
			panic(err)
		}
		code := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random[:])
		passes[i] = fmt.Sprintf("P%02d-%s", i+1, code)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(passes); err != nil {
		panic(err)
	}
}
