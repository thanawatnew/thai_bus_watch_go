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
	check := flag.String("check", "", "validate one pass without printing the pass database")
	file := flag.String("file", "priority-passes.json", "pass database used by -check")
	flag.Parse()
	if *check != "" {
		body, err := os.ReadFile(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot read pass file:", err)
			os.Exit(2)
		}
		var passes []string
		if err := json.Unmarshal(body, &passes); err != nil {
			fmt.Fprintln(os.Stderr, "invalid pass file:", err)
			os.Exit(2)
		}
		for index, pass := range passes {
			if pass == *check {
				fmt.Printf("valid priority pass; rank=%d of %d\n", index+1, len(passes))
				return
			}
		}
		fmt.Fprintln(os.Stderr, "invalid priority pass")
		os.Exit(1)
	}
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
