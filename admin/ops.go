package admin

import (
	"bytes"
	"embed"
	"encoding/gob"
	"github.com/fabricekabongo/loggerhead/clustering"
	"github.com/fabricekabongo/loggerhead/world"
	"github.com/hashicorp/memberlist"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"html/template"
	"net/http"
)

var (
	//go:embed template
	TemplateFS embed.FS
	//go:embed static
	StaticFS  embed.FS
	AdminTMPL *template.Template
)

func init() {
	tmpl, err := template.ParseFS(TemplateFS, "template/admin.html")
	if err != nil {
		panic(err)
	}

	AdminTMPL = tmpl
}

type OpsServer struct {
	mList *memberlist.Memberlist
	world *world.Map
}

func NewOpsServer(mList *memberlist.Memberlist, world *world.Map) *OpsServer {
	return &OpsServer{mList: mList, world: world}
}

func (o *OpsServer) Start() {

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(StaticFS))))
	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", o.AdminUI())
	err := http.ListenAndServe(":20000", nil)
	if err != nil {
		return
	}

}

type AdminData struct {
	NodesAlive int
	Members    []Member
	Locations  int
	Grids      int
}

type Member struct {
	Name  string
	Addr  string
	State string
	Meta  clustering.NodeMetaData
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
func (o *OpsServer) AdminUI() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		members := o.mList.Members()

		data := AdminData{
			NodesAlive: o.mList.NumMembers(),
			Members:    make([]Member, 0, len(members)),
		}
		for _, member := range members {
			dec := gob.NewDecoder(bytes.NewReader(member.Meta))
			var meta clustering.NodeMetaData
			err := dec.Decode(&meta)
			if err != nil {
				meta = clustering.NodeMetaData{}
			}

			data.Members = append(data.Members, Member{
				Name:  member.Name,
				Addr:  member.Addr.String(),
				State: stateToString(member.State),
				Meta:  meta,
			})
		}
		err := AdminTMPL.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
