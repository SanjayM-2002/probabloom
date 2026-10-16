// Shared by cmd/probabloom and cmd/probabloom-lite; the binaries differ only
// in which backends they import.
package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"

	"github.com/SanjayM-2002/probabloom"
	"github.com/SanjayM-2002/probabloom/internal/experiment"
	"github.com/SanjayM-2002/probabloom/internal/sizing"
	"github.com/SanjayM-2002/probabloom/registry"
	"github.com/SanjayM-2002/probabloom/strategy/counting"
	"github.com/SanjayM-2002/probabloom/username"
)

type app struct {
	ctx    context.Context
	cfg    registry.Config
	f      probabloom.Filter
	loaded []string // names from -load, also used as the database for "available"
	dirty  bool
}

func Main() {
	log.SetFlags(0)

	backend := flag.String("backend", "memory", fmt.Sprintf("storage backend %v", registry.Backends()))
	strategy := flag.String("strategy", "standard", fmt.Sprintf("filter strategy %v (memory only)", registry.Strategies()))
	hashName := flag.String("hash", "xxh64", fmt.Sprintf("hash function %v (memory only)", registry.Hashes()))
	positions := flag.String("positions", "double", fmt.Sprintf("how the k positions are derived %v (memory only)", registry.Positions()))
	n := flag.Uint64("n", 1_000_000, "expected number of items")
	p := flag.Float64("p", 0.01, "target false-positive rate")
	load := flag.String("load", "", "file of names (one per line) to add before running the command")
	file := flag.String("file", "", "save the filter here and load it on the next run (memory only)")
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis address (redisbloom only)")
	redisDB := flag.Int("redis-db", 0, "Redis database number (redisbloom only)")
	key := flag.String("key", "probabloom:usernames", "Redis key (redisbloom only)")
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}

	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	cfg := registry.Config{
		Backend: *backend, Strategy: *strategy, Hash: *hashName, Positions: *positions, N: *n, P: *p,
		RedisAddr: *redisAddr, RedisDB: *redisDB, RedisKey: *key,
	}
	if cfg.Backend != "memory" {
		// these defaults only make sense for memory; an explicit flag is still passed on (and rejected)
		if !set["strategy"] {
			cfg.Strategy = ""
		}
		if !set["hash"] {
			cfg.Hash = ""
		}
		if !set["positions"] {
			cfg.Positions = ""
		}
	}

	a := &app{ctx: context.Background(), cfg: cfg}
	if err := a.open(*file, set); err != nil {
		log.Fatal(err)
	}
	if *load != "" {
		if err := a.loadNames(*load); err != nil {
			log.Fatal(err)
		}
	}

	args := flag.Args()[1:]
	var err error
	switch cmd := flag.Arg(0); cmd {
	case "info":
		err = a.info()
	case "add":
		err = a.add(args)
	case "check":
		err = a.check(args)
	case "remove":
		err = a.remove(args)
	case "available":
		err = a.available(args)
	case "repl":
		err = a.repl(os.Stdin)
	case "fp":
		err = a.fp()
	case "bench":
		err = bench(a.ctx, cfg)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err == nil && *file != "" && a.dirty {
		if err = registry.SaveFile(*file, a.cfg, a.f); err == nil {
			fmt.Printf("saved to %s\n", *file)
		}
	}
	if c, ok := a.f.(io.Closer); ok {
		c.Close()
	}
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage: probabloom [flags] <command> [args]

commands:
  info              show the filter's size and shape
  add NAME...       add names
  check NAME...     check names against the filter only
  remove NAME...    remove names (counting strategy only)
  available NAME... full username check: filter first, then the database
                    (the database is the -load file)
  repl              interactive: "add NAME", "check NAME", "remove NAME"
  fp                add n items, query n absent items, report the real
                    false-positive rate and speed
  bench             fp for every strategy × hash × positions (memory)

names are normalized first (trimmed, NFKC, lowercased): "Robin" == " robin "

the memory backend forgets everything on exit unless -file is given;
redisbloom keeps its data in Redis.

flags:
`)
	flag.PrintDefaults()
}

// With -file, an existing file decides the settings; flags that disagree
// with it are an error rather than silently ignored.
func (a *app) open(path string, set map[string]bool) error {
	if path == "" {
		f, err := registry.New(a.ctx, a.cfg)
		a.f = f
		return err
	}
	if a.cfg.Backend != "memory" {
		return fmt.Errorf("-file only works with -backend=memory (redisbloom already persists in Redis)")
	}

	f, saved, err := registry.LoadFile(a.ctx, path)
	if errors.Is(err, fs.ErrNotExist) {
		a.f, err = registry.New(a.ctx, a.cfg)
		a.dirty = true
		return err
	}
	if err != nil {
		return fmt.Errorf("loading %s: %w", path, err)
	}

	conflicts := map[string][2]string{
		"strategy":  {a.cfg.Strategy, saved.Strategy},
		"hash":      {a.cfg.Hash, saved.Hash},
		"positions": {a.cfg.Positions, saved.Positions},
		"n":         {fmt.Sprint(a.cfg.N), fmt.Sprint(saved.N)},
		"p":         {fmt.Sprint(a.cfg.P), fmt.Sprint(saved.P)},
	}
	for name, v := range conflicts {
		if set[name] && v[0] != v[1] {
			return fmt.Errorf("%s was created with -%s=%s; drop -%s or use a different -file", path, name, v[1], name)
		}
	}
	a.f, a.cfg = f, saved
	return nil
}

func (a *app) loadNames(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	sc := bufio.NewScanner(file)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		name, err := username.Normalize(sc.Text())
		if err != nil {
			return fmt.Errorf("%s: %q: %w", path, sc.Text(), err)
		}
		if err := a.f.Add(a.ctx, []byte(name)); err != nil {
			return err
		}
		a.loaded = append(a.loaded, name)
	}
	if err := sc.Err(); err != nil {
		return err
	}
	a.dirty = true
	fmt.Printf("loaded %d names from %s\n", len(a.loaded), path)
	return nil
}

func (a *app) info() error {
	info, err := registry.Describe(a.ctx, a.f)
	if err != nil {
		return err
	}
	c := a.cfg
	if c.Backend == "memory" {
		fmt.Printf("backend=memory strategy=%s hash=%s positions=%s\n", c.Strategy, c.Hash, c.Positions)
	} else {
		fmt.Printf("backend=%s key=%s (strategy and hash chosen by Redis)\n", c.Backend, c.RedisKey)
	}
	fmt.Printf("n=%d p=%v", c.N, c.P)
	if info.M > 0 {
		fmt.Printf(" → m=%d bits, k=%d hashes (%.2f bits/item)", info.M, info.K, float64(info.M)/float64(c.N))
	}
	fmt.Printf("\nmemory %s, %d layer(s)\n", size(info.SizeBytes), info.Layers)
	return nil
}

func (a *app) add(names []string) error {
	return eachName(names, func(name string) error {
		if err := a.f.Add(a.ctx, []byte(name)); err != nil {
			return err
		}
		a.dirty = true
		fmt.Printf("%-20s added\n", name)
		return nil
	})
}

func (a *app) check(names []string) error {
	return eachName(names, func(name string) error {
		ok, err := a.f.Contains(a.ctx, []byte(name))
		if err != nil {
			return err
		}
		verdict := "definitely not added"
		if ok {
			verdict = "maybe added"
		}
		fmt.Printf("%-20s %s\n", name, verdict)
		return nil
	})
}

func (a *app) remove(names []string) error {
	r, ok := a.f.(probabloom.Remover)
	if !ok {
		return fmt.Errorf("this filter can't remove items (use -strategy=counting)")
	}
	return eachName(names, func(name string) error {
		err := r.Remove(a.ctx, []byte(name))
		switch {
		case errors.Is(err, counting.ErrNotPresent):
			fmt.Printf("%-20s not present, nothing removed\n", name)
		case err != nil:
			return err
		default:
			a.dirty = true
			fmt.Printf("%-20s removed\n", name)
		}
		return nil
	})
}

func (a *app) available(names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("available needs at least one NAME")
	}
	if a.loaded == nil {
		fmt.Println("(no -load file, so the database is empty)")
	}
	db := username.NewMemoryDB(a.loaded...)
	c := username.NewChecker(a.f, db)
	for _, raw := range names {
		res, err := c.Check(a.ctx, raw)
		if err != nil {
			return fmt.Errorf("%q: %w", raw, err)
		}
		switch {
		case !res.CheckedDB:
			fmt.Printf("%-20s available   filter: definitely not taken (no database lookup)\n", res.Name)
		case res.Available:
			fmt.Printf("%-20s available   filter: maybe → database: not found (false positive)\n", res.Name)
		default:
			fmt.Printf("%-20s taken       filter: maybe → database: found\n", res.Name)
		}
	}
	return nil
}

func (a *app) repl(in io.Reader) error {
	help := `commands: "add NAME", "check NAME", ctrl+D to quit`
	if _, ok := a.f.(probabloom.Remover); ok {
		help = `commands: "add NAME", "check NAME", "remove NAME", ctrl+D to quit`
	}
	fmt.Println(help)

	sc := bufio.NewScanner(in)
	for fmt.Print("> "); sc.Scan(); fmt.Print("> ") {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 {
			fmt.Println(help)
			continue
		}
		var err error
		switch fields[0] {
		case "add":
			err = a.add(fields[1:])
		case "check":
			err = a.check(fields[1:])
		case "remove":
			err = a.remove(fields[1:])
		default:
			fmt.Println(help)
		}
		if err != nil {
			fmt.Println("error:", err)
		}
	}
	fmt.Println()
	return sc.Err()
}

func (a *app) fp() error {
	if err := a.info(); err != nil {
		return err
	}
	if a.cfg.Backend != "memory" {
		fmt.Printf("(%d network round trips, this can take a while; lower -n to speed up)\n", 2*a.cfg.N)
	}
	n := int(a.cfg.N)
	res, err := experiment.Run(a.ctx, a.f, experiment.Keys("added", n), experiment.Keys("absent", n))
	if err != nil {
		return err
	}
	fmt.Printf("target    %.4f%%\n", a.cfg.P*100)
	if info, _ := registry.Describe(a.ctx, a.f); info.M > 0 {
		expected := sizing.EstimatedFPRate(sizing.Params{M: info.M, K: info.K}, a.cfg.N)
		fmt.Printf("formula   %.4f%%  (standard bloom formula for this m and k)\n", expected*100)
	}
	fmt.Printf("measured  %.4f%%  (%d false positives / %d queries)\n", res.Rate()*100, res.FalsePositives, res.Queried)
	fmt.Printf("speed     add %.0f ns, contains %.0f ns\n", res.AddNs(), res.ContainsNs())
	return nil
}

func bench(ctx context.Context, base registry.Config) error {
	n := int(base.N)
	added, absent := experiment.Keys("added", n), experiment.Keys("absent", n)

	fmt.Printf("n=%d p=%v, memory backend\n\n", base.N, base.P)
	fmt.Printf("%-12s %-8s %-9s %9s %12s %9s %10s\n", "strategy", "hash", "positions", "add ns", "contains ns", "fp", "memory")
	for _, s := range registry.Strategies() {
		for _, h := range registry.Hashes() {
			for _, pos := range registry.Positions() {
				cfg := registry.Config{Backend: "memory", Strategy: s, Hash: h, Positions: pos, N: base.N, P: base.P}
				f, err := registry.New(ctx, cfg)
				if err != nil {
					return err
				}
				res, err := experiment.Run(ctx, f, added, absent)
				if err != nil {
					return err
				}
				info, _ := registry.Describe(ctx, f)
				fmt.Printf("%-12s %-8s %-9s %9.1f %12.1f %8.3f%% %10s\n",
					s, h, pos, res.AddNs(), res.ContainsNs(), res.Rate()*100, size(info.SizeBytes))
			}
		}
	}
	return nil
}

func eachName(raw []string, fn func(name string) error) error {
	if len(raw) == 0 {
		return fmt.Errorf("needs at least one NAME")
	}
	for _, r := range raw {
		name, err := username.Normalize(r)
		if err != nil {
			return fmt.Errorf("%q: %w", r, err)
		}
		if err := fn(name); err != nil {
			return err
		}
	}
	return nil
}

func size(b uint64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.2f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	}
	return fmt.Sprintf("%d B", b)
}
