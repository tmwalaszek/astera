package main

import (
	"astera/handler"
	"astera/modstore"
	"astera/sqlite3"
	"flag"
	"log"
	"os"
	"strconv"

	"net/http"
	_ "net/http/pprof"
)

func envString(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		log.Fatalf("invalid bool for %s=%q: %v", key, v, err)
	}
	return b
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	dbName := flag.String("db", envString("ASTERA_DB", "astera.db"), "database file (env: ASTERA_DB)")
	pprofEnable := flag.Bool("pprof", envBool("ASTERA_PPROF", false), "enable pprof (env: ASTERA_PPROF)")
	importLocalCache := flag.Bool("import-local-cache", envBool("ASTERA_IMPORT_LOCAL_CACHE", false), "import local cache (env: ASTERA_IMPORT_LOCAL_CACHE)")
	localCacheDir := flag.String("local-cache-dir", envString("ASTERA_LOCAL_CACHE_DIR", homeDir+"/go/pkg/mod/cache/download"), "local cache directory (env: ASTERA_LOCAL_CACHE_DIR)")
	addr := flag.String("addr", envString("ASTERA_ADDR", ":8080"), "listen address (env: ASTERA_ADDR)")

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
