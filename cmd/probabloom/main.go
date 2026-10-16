package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/experiment"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/registry"
	"github.com/SanjayM-2002/probabloom/strategy/counting"
)

func main() {
	log.SetFlags(0)

	backend := flag.String("backend", "memory", fmt.Sprintf("storage backend %v", registry.Backends()))
	strategy := flag.String("strategy", "standard", fmt.Sprintf("filter strategy %v", registry.Strategies()))
	hashName := flag.String("hash", "fnv1a", fmt.Sprintf("hash function %v", registry.Hashes()))
	positions := flag.String("positions", "seeded", fmt.Sprintf("how the k positions are derived %v", registry.Positions()))
	n := flag.Uint64("n", 1_000_000, "expected number of items")
	p := flag.Float64("p", 0.01, "target false-positive rate")
	load := flag.String("load", "", "file of names (one per line) to add before running the command")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}

	ctx := context.Background()
	cfg := registry.Config{Backend: *backend, Strategy: *strategy, Hash: *hashName, Positions: *positions, N: *n, P: *p}
	f, info, err := registry.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}

	if *load != "" {
		count, err := loadFile(ctx, f, *load)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("loaded %d names from %s\n", count, *load)
	}

	switch cmd := flag.Arg(0); cmd {
	case "info":
		printInfo(cfg, info)
	case "check":
		err = check(ctx, f, flag.Args()[1:])
	case "repl":
		err = repl(ctx, f, os.Stdin)
	case "fp":
		err = fp(ctx, f, cfg, info)
	default:
		log.Fatalf("unknown command %q", cmd)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage: probabloom [flags] <command> [args]

commands:
  info          show filter size (m bits, k hashes, memory)
  check NAME... check names (combine with -load)
  repl          interactive: "add NAME", "check NAME", "remove NAME" (counting only)
  fp            add n items, query n absent items, report false-positive rate

flags:
`)
	flag.PrintDefaults()
}

func printInfo(cfg registry.Config, info registry.Info) {
	fmt.Printf("backend=%s strategy=%s hash=%s positions=%s\n", cfg.Backend, cfg.Strategy, cfg.Hash, cfg.Positions)
	fmt.Printf("n=%d p=%v → m=%d bits, k=%d hashes, %.2f MB (%.2f bits/item)\n",
		cfg.N, cfg.P, info.M, info.K, float64(info.SizeBytes)/(1<<20), float64(info.M)/float64(cfg.N))
}

func check(ctx context.Context, f probabloom.Filter, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("check needs at least one NAME")
	}
	for _, name := range names {
		ok, err := f.Contains(ctx, []byte(name))
		if err != nil {
			return err
		}
		fmt.Printf("%-20s %s\n", name, verdict(ok))
	}
	return nil
}

func repl(ctx context.Context, f probabloom.Filter, in io.Reader) error {
	help := `commands: "add NAME", "check NAME", ctrl+D to quit`
	remover, canRemove := f.(probabloom.Remover)
	if canRemove {
		help = `commands: "add NAME", "check NAME", "remove NAME", ctrl+D to quit`
	}
	fmt.Println(help)
	sc := bufio.NewScanner(in)
	for fmt.Print("> "); sc.Scan(); fmt.Print("> ") {
		fields := strings.Fields(sc.Text())
		switch {
		case len(fields) == 0:
		case len(fields) == 2 && fields[0] == "add":
			if err := f.Add(ctx, []byte(fields[1])); err != nil {
				return err
			}
			fmt.Println("added")
		case len(fields) == 2 && fields[0] == "check":
			if err := check(ctx, f, fields[1:]); err != nil {
				return err
			}
		case len(fields) == 2 && fields[0] == "remove" && canRemove:
			err := remover.Remove(ctx, []byte(fields[1]))
			switch {
			case errors.Is(err, counting.ErrNotPresent):
				fmt.Println("not present, nothing removed")
			case err != nil:
				return err
			default:
				fmt.Println("removed")
			}
		case len(fields) == 2 && fields[0] == "remove":
			fmt.Println("this strategy does not support remove (use -strategy=counting)")
		default:
			fmt.Println(help)
		}
	}
	fmt.Println()
	return sc.Err()
}

func fp(ctx context.Context, f probabloom.Filter, cfg registry.Config, info registry.Info) error {
	printInfo(cfg, info)
	res, err := experiment.MeasureFP(ctx, f, int(cfg.N))
	if err != nil {
		return err
	}
	expected := sizing.EstimatedFPRate(sizing.Params{M: info.M, K: info.K}, cfg.N)
	fmt.Printf("target    %.4f%%\n", cfg.P*100)
	fmt.Printf("formula   %.4f%%  (standard bloom formula for this m and k)\n", expected*100)
	fmt.Printf("measured  %.4f%%  (%d false positives / %d queries)\n", res.Rate()*100, res.FalsePositives, res.Queried)
	return nil
}

func loadFile(ctx context.Context, f probabloom.Filter, path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		name := strings.TrimSpace(sc.Text())
		if name == "" {
			continue
		}
		if err := f.Add(ctx, []byte(name)); err != nil {
			return count, err
		}
		count++
	}
	return count, sc.Err()
}

func verdict(maybe bool) string {
	if maybe {
		return "maybe taken"
	}
	return "available (definitely not added)"
}
