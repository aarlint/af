package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/aarlint/af/internal/db"
	"github.com/aarlint/af/internal/models"
)

func main() {
	dbPath := flag.String("db", "af.db", "path to SQLite database")
	dataDir := flag.String("data", "seed/data", "directory containing seed JSON files")
	flag.Parse()

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer database.Close()

	if err := db.Init(database); err != nil {
		log.Fatalf("init db: %v", err)
	}

	files, err := filepath.Glob(filepath.Join(*dataDir, "*.json"))
	if err != nil {
		log.Fatalf("glob: %v", err)
	}
	sort.Strings(files)

	totalActions := 0
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("read %s: %v", f, err)
		}

		var seed models.SeedFile
		if err := json.Unmarshal(data, &seed); err != nil {
			log.Fatalf("parse %s: %v", f, err)
		}

		tx, err := database.Begin()
		if err != nil {
			log.Fatalf("begin tx: %v", err)
		}

		for _, a := range seed.Actions {
			actionID, err := db.InsertAction(tx, seed.President, a.EO, a.Title, a.Date, a.Category, a.URL)
			if err != nil {
				tx.Rollback()
				log.Fatalf("insert action %q: %v", a.Title, err)
			}

			for country, score := range a.Impacts {
				reason := a.Reasons[country]
				if err := db.InsertImpact(tx, actionID, country, score, reason); err != nil {
					tx.Rollback()
					log.Fatalf("insert impact for action %q country %s: %v", a.Title, country, err)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			log.Fatalf("commit: %v", err)
		}

		totalActions += len(seed.Actions)
		fmt.Printf("Seeded %s: %d actions from %s\n", seed.President, len(seed.Actions), filepath.Base(f))
	}

	fmt.Printf("Total: %d actions seeded\n", totalActions)
}
