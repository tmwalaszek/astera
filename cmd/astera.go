package main

import (
	"astera/handler"
	"astera/modstore"
	"astera/sqlite3"
	"flag"
	"log"
	"os"

	"net/http"
	_ "net/http/pprof"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	dbName := flag.String("db", "astera.db", "database file")
	pprofEnable := flag.Bool("pprof", false, "enable pprof")
	importLocalCache := flag.Bool("import-local-cache", false, "import local cache")
	localCacheDir := flag.String("local-cache-dir", homeDir+"/go/pkg/mod/cache/download", "local cache directory")
	addr := flag.String("addr", ":8080", "listen address")

	flag.Parse()

	db, err := sqlite3.NewDB(*dbName)
	if err != nil {
		panic(err)
	}

	if *pprofEnable {
		go func() {
			log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
		}()
	}

	m := modstore.NewModuleStore(db)
	if *importLocalCache {
		err = m.ImportCachedModules(*localCacheDir)
		if err != nil {
			panic(err)
		}
	}

	h := handler.NewHandler(m)

	mux := http.NewServeMux()
	mux.Handle("/", handler.LoggerMiddlerware(h))

	err = http.ListenAndServe(*addr, mux)
	if err != nil {
		panic(err)
	}
}
