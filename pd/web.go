package pd

import (
	"encoding/json"
	"fmt"
	"net/http"
	//"net/http/pprof"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/zssky/log"

	"github.com/taorenhai/ancestor/meta"
)

var (
	server *Server
)

var routerMap = map[string]map[string]func(http.ResponseWriter, *http.Request){
	"GET": {
		"/api/stats":      getStats,
		"/api/node":       getNodes,
		"/api/node/{id}":  getNodeInfo,
		"/api/range":      getRanges,
		"/api/range/{id}": getRangeInfo,
	},

	"POST": {},

	"DELETE": {
		"/range/{id}": delRange,
	},
}

func newWebRouter() http.Handler {
	router := mux.NewRouter()

	for method, routes := range routerMap {
		for route, funcHandler := range routes {
			router.Path(route).Methods(method).HandlerFunc(funcHandler)
		}
	}

	router.NotFoundHandler = http.NotFoundHandler()
	return router
}

func delRange(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func getStats(w http.ResponseWriter, r *http.Request) {
	ns := server.cluster.getNodeStats()

	type response struct {
		Nodes  int
		Keys   int64
		Bytes  int64
		Ranges int
	}

	resp := response{}

	for i, n := range ns {
		resp.Nodes++
		if i == 0 {
			resp.Bytes = n.TotalBytes
			resp.Keys = n.TotalCount
		}
	}

	rds, err := server.region.getRangeDescriptors(nil)
	if err != nil {
		fmt.Fprintf(w, "getRangeDescriptors error:%s", err.Error())
		return
	}

	resp.Ranges = len(rds)

	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(w, "json Marshal error:%s", err.Error())
		return
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

func getNodes(w http.ResponseWriter, r *http.Request) {
	type nodeInfo struct {
		ID       meta.NodeID
		Address  string
		Capacity int64
		Leader   uint64
		Ranges   int
		Bytes    int64
	}
	var nodes []nodeInfo

	ns := server.cluster.getNodes()
	for _, n := range ns {
		node := nodeInfo{ID: n.NodeID, Address: n.Address, Capacity: n.Capacity}
		stats := server.cluster.getNodeStats(n.NodeID)
		if len(stats) != 0 {
			node.Leader = stats[0].LeaderCount
			node.Ranges = len(stats[0].RangeStatsInfo)
			node.Bytes = stats[0].TotalBytes
		}
		nodes = append(nodes, node)
	}

	data, err := json.Marshal(nodes)
	if err != nil {
		fmt.Fprintf(w, "json Marshal error:%s", err.Error())
		return
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

func getNodeInfo(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("show OK\n"))
}

func getRanges(w http.ResponseWriter, r *http.Request) {
	ranges, err := server.region.getRangeDescriptors(nil)
	if err != nil {
		fmt.Fprintf(w, "getRangeDescriptors error:%s", err.Error())
		return
	}

	type rangeInfo struct {
		RangeID  int
		StartKey string
		EndKey   string
		Bytes    int64
		Keys     int64
		Percent  int
	}

	var rds []rangeInfo

	for _, r := range ranges {
		rs := server.cluster.getRangeStats(r.RangeID)
		ri := rangeInfo{RangeID: int(r.RangeID), StartKey: r.StartKey.String(), EndKey: r.EndKey.String()}

		if len(rs) != 0 {
			ri.Bytes = rs[0].TotalBytes
			ri.Keys = rs[0].TotalCount
		}

		ri.Percent = int(ri.Bytes * 100 / int64(server.cfg.RangeCapacity))
		rds = append(rds, ri)
	}

	data, err := json.Marshal(rds)
	if err != nil {
		fmt.Fprintf(w, "json Marshal error:%s", err.Error())
		return
	}

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

func getRangeInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		fmt.Fprintf(w, "strconv id error:%s", err.Error())
		return
	}

	rds := server.cluster.getRangeStats(meta.RangeID(id))
	data, err := json.Marshal(rds)
	if err != nil {
		fmt.Fprintf(w, "json Marshal error:%s", err.Error())
		return
	}
	log.Debugf("%s\n", data)

	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

func startWebAPI(s *Server) {
	server = s
	http.Handle("/", http.FileServer(http.Dir(s.cfg.WebPath)))
	http.Handle("/api/", newWebRouter())
	//	http.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))

	s.stopper.RunWorker(func() {
		if err := http.ListenAndServe(s.cfg.WebHost, nil); err != nil {
			log.Errorf("http service error:%s", err.Error())
		}
	})

}
