package admin

import (
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/fabricekabongo/loggerhead/clustering"
	"github.com/fabricekabongo/loggerhead/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	//go:embed template
	TemplateFS embed.FS
	//go:embed static
	StaticFS embed.FS
	TMPL     *template.Template
)

func init() {
	tmpl, err := template.ParseFS(TemplateFS, "template/admin.html")
	if err != nil {
		panic(err)
	}

	TMPL = tmpl
}

type OpsServer struct {
	cluster *clustering.Cluster
	cfg     config.Config
}

func NewOpsServer(cluster *clustering.Cluster, cfg config.Config) *OpsServer {
	return &OpsServer{
		cluster: cluster,
		cfg:     cfg,
	}
}

func (o *OpsServer) Start() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(StaticFS))))
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/admin-data", o.AdminData())
	http.Handle("/", o.AdminUI())

	server := &http.Server{
		Addr:              ":20000",
		ReadHeaderTimeout: 3 * time.Second,
	}

	err := server.ListenAndServe()
	if err != nil {
		log.Println("Failed to start the admin server: ", err)
		return
	}
}

type Data struct {
	Name       string
	NodesAlive int
	Health     int
	MemStats   MemStats
	CPUs       int
	GoRoutines int
	Others     []Data
	State      string
	Address    string
	QueueCount int
}

type MemStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
}

func (o *OpsServer) AdminData() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var memStats runtime.MemStats

		runtime.ReadMemStats(&memStats)

		data := Data{
			Name:       o.cluster.MemberList().LocalNode().Name,
			Address:    o.cluster.MemberList().LocalNode().Addr.String(),
			NodesAlive: o.cluster.MemberList().NumMembers(),
			MemStats: MemStats{
				Alloc:      (memStats.Alloc / 1024) / 1024,
				TotalAlloc: (memStats.TotalAlloc / 1024) / 1024,
				Sys:        (memStats.Sys / 1024) / 1024,
			},
			CPUs:       runtime.NumCPU(),
			GoRoutines: runtime.NumGoroutine(),
			Health:     o.cluster.MemberList().GetHealthScore(),
			State:      clustering.StateToString(o.cluster.MemberList().LocalNode().State),
			QueueCount: o.cluster.Broadcasts().NumQueued(),
		}

		getParams := r.URL.Query()
		if getParams.Get("proxy") != "true" {
			members := o.cluster.MemberList().Members()
			membersAdminData := make([]Data, 0, len(members))

			for _, member := range members {
				if member.Name == o.cluster.MemberList().LocalNode().Name {
					continue
				}
				httpResp, err := http.Get("http://" + member.Addr.String() + ":20000/admin-data?proxy=true")
				if err != nil {
					continue
				}

				var memberData Data
				err = json.NewDecoder(httpResp.Body).Decode(&memberData)
				if err != nil {
					continue
				}

				membersAdminData = append(membersAdminData, memberData)
			}

			data.Others = membersAdminData
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func (*OpsServer) AdminUI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		err := TMPL.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
