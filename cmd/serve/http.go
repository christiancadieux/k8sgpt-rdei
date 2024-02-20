package serve

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/k8sgpt-ai/k8sgpt/pkg/analysis"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/k8sgpt-ai/k8sgpt/pkg/ai"
	k8sgptserver "github.com/k8sgpt-ai/k8sgpt/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var Server k8sgptserver.Config

var HttpCmd = &cobra.Command{
	Use:   "http",
	Short: "Runs k8sgpt webserver",
	Long:  `Runs k8sgpt webserver.`,
	Run: func(cmd *cobra.Command, args []string) {

		var configAI ai.AIConfiguration
		err := viper.UnmarshalKey("ai", &configAI)
		if err != nil {
			color.Red("Error: %v", err)
			os.Exit(1)
		}
		var aiProvider *ai.AIProvider

		os.Setenv("REMOTE", "Y")
		if aiProvider == nil {
			for _, provider := range configAI.Providers {
				if backend == provider.Name {
					// the pointer to the range variable is not really an issue here, as there
					// is a break right after, but to prevent potential future issues, a temp
					// variable is assigned
					p := provider
					aiProvider = &p
					break
				}
			}
		}

		if aiProvider.Name == "" {
			color.Red("Error: AI provider %s not specified in configuration. Please run k8sgpt auth", backend)
			os.Exit(1)
		} else {
			fmt.Println("AI provider", aiProvider.Name)
		}

		logger, err := zap.NewProduction()
		if err != nil {
			color.Red("failed to create logger: %v", err)
			os.Exit(1)
		}
		defer logger.Sync()

		Server = k8sgptserver.Config{
			Backend:     aiProvider.Name,
			Port:        port,
			MetricsPort: metricsPort,
			Token:       aiProvider.Password,
			Logger:      logger,
		}

		r := mux.NewRouter()
		// new format: /api/gpt?cl=..&ns=..
		// old format: /refresh?cl=..&ns=..
		// static format: /{cl}_{ns}.json     # will not be needed after rdei-core is updated
		api := r.PathPrefix("/api/").Subrouter()
		api.HandleFunc("/gpt", GetGpt).Methods("GET")

		r.HandleFunc("/refresh", GetGpt).Methods("GET")

		r.HandleFunc("/{id}", GetGptId).Methods("GET")

		err = analysis.LoadResolveIndex()
		if err != nil {
			fmt.Println("loadResolve err", err)
			return
		}

		fmt.Println("Start http.Server port=", port)
		srv := &http.Server{
			Handler:      access_log(r), // handlers.LoggingHandler(os.Stdout, r),
			Addr:         ":" + port,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second}

		log.Fatal(srv.ListenAndServe())

	},
}

var output string

func GetGptId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	parts := strings.Split(id, "_")
	if len(parts) < 2 {
		http.Error(w, "invalid argument", 400)
		return
	}

	ns := strings.Replace(parts[1], ".json", "", 1)
	cs := parts[0]
	getGpt(w, r, cs, ns)
}

func GetGpt(w http.ResponseWriter, r *http.Request) {

	cluster := r.URL.Query().Get("cl")
	namespace := r.URL.Query().Get("ns")
	if cluster == "" || namespace == "" {
		http.Error(w, "cl=cluster or ns=namespace missing", 400)
		return
	}
	getGpt(w, r, cluster, namespace)
}

func getGpt(w http.ResponseWriter, r *http.Request, cluster, namespace string) {

	explain := false
	if r.URL.Query().Get("explain") != "" {
		explain = true
	}
	cache := false
	if r.URL.Query().Get("cache") != "" {
		cache = true
	}
	language := "english"

	docs := true
	if r.URL.Query().Get("nodocs") != "" {
		docs = false
	}

	filters := []string{}
	filter := r.URL.Query().Get("filters")
	if filter != "" {
		filters = append(filters, filter)
	}
	fmt.Printf("NewAnalysis backend=%s, cluster=%s, ns=%s, cache=%v, docs=%v, filter=%v \n",
		backend, cluster, namespace, cache, docs, filters)

	config, err := analysis.NewAnalysis(backend,
		language, filters, namespace, !cache, explain, 10, docs,
		cluster)
	if err != nil {
		http.Error(w, err.Error(), 400)
	}

	config.RunAnalysis()

	err = config.GetResolutionText(output)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	// print results
	output, err := config.PrintOutput("json")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Write(output)
}

func access_log(r http.Handler) http.Handler {
	f, err := os.OpenFile("./access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Panic("Access log: ", err)
	}

	return handlers.LoggingHandler(f, r)
}

func init() {
	// add flag for backend
	ServeCmd.Flags().StringVarP(&port, "port", "p", "8888", "Port to run the server on")
	// ServeCmd.Flags().StringVarP(&backend, "backend", "b", "openai", "Backend AI provider")
}
