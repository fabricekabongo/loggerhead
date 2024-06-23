package admin

import (
	"embed"
	"encoding/json"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"html/template"
	"log"
	"net/http"
	"runtime"
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
	mList *memberlist.Memberlist
	world *world.World
}

func NewOpsServer(mList *memberlist.Memberlist, world *world.World) *OpsServer {
	return &OpsServer{mList: mList, world: world}
}

func (o *OpsServer) Start() {

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(StaticFS))))
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/admin-data", o.AdminData())
	http.Handle("/", o.AdminUI())
	err := http.ListenAndServe(":20000", nil)
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
}

type MemStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
}

func stateToString(state memberlist.NodeStateType) string {
	switch state {
	case memberlist.StateAlive:
		return "Alive"
	case memberlist.StateSuspect:
		return "Suspect"
	case memberlist.StateLeft:
		return "Left"
	case memberlist.StateDead:
		return "Dead"
	}

	return "Unknown"
}
func (o *OpsServer) AdminData() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var memStats runtime.MemStats

		runtime.ReadMemStats(&memStats)

		data := Data{
			Name:       o.mList.LocalNode().Name,
			Address:    o.mList.LocalNode().Addr.String(),
			NodesAlive: o.mList.NumMembers(),
			MemStats: MemStats{
				Alloc:      (memStats.Alloc / 1024) / 1024,
				TotalAlloc: (memStats.TotalAlloc / 1024) / 1024,
				Sys:        (memStats.Sys / 1024) / 1024,
			},
			CPUs:       runtime.NumCPU(),
			GoRoutines: runtime.NumGoroutine(),
			Health:     o.mList.GetHealthScore(),
			State:      stateToString(o.mList.LocalNode().State),
		}

		getParams := r.URL.Query()
		if getParams.Get("proxy") != "true" {
			members := o.mList.Members()
			membersAdminData := make([]Data, 0, len(members))

			for _, member := range members {
				if member.Name == o.mList.LocalNode().Name {
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
		}
	})
}

func (o *OpsServer) AdminUI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := TMPL.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
