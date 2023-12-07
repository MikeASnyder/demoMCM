package catalog

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rancher/rancher/pkg/controllers/dashboard/plugin"
	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/endpoints/request"
	"net/http"
	"net/http/httputil"
	neturl "net/url"
	"strings"
)

func SetupUIPluginHandlers(router *mux.Router) {
	router.HandleFunc("/v1/uiplugins", indexHandler)
	router.HandleFunc("/v1/uiplugins/{name}/{version}/{rest:.*}", pluginHandler)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	authed := isAuthenticated(r)
	if authed {
		index, err := json.Marshal(&plugin.Index)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logrus.Error(err)
		}
		w.Write(index)
	} else {
		index, err := json.Marshal(&plugin.AnonymousIndex)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logrus.Error(err)
		}
		w.Write(index)
	}
}

func pluginHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logrus.Debugf("http request vars %s", vars)
	authed := isAuthenticated(r)
	entry, ok := plugin.Index.Entries[vars["name"]]
	if (!ok || entry.Version != vars["version"]) || (!authed && !entry.NoAuth) {
		msg := fmt.Sprintf("plugin [name: %s version: %s] does not exist in index", vars["name"], vars["version"])
		http.Error(w, msg, http.StatusNotFound)
		logrus.Debug(msg)
		return
	}
	if entry.NoCache {
		logrus.Debugf("[noCache: %v] proxying request to [endpoint: %v]\n", entry.NoCache, entry.Endpoint)
		proxyRequest(entry.Endpoint, vars["rest"], w, r)
	} else {
		logrus.Debugf("[noCache: %v] serving plugin files from filesystem cache\n", entry.NoCache)
		r.URL.Path = fmt.Sprintf("/%s/%s/%s", vars["name"], vars["version"], vars["rest"])
		http.FileServer(http.Dir(plugin.FSCacheRootDir)).ServeHTTP(w, r)
	}
}

func proxyRequest(target, path string, w http.ResponseWriter, r *http.Request) {
	url, err := neturl.Parse(target)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse url [%s]", target), http.StatusInternalServerError)
		return
	}
	if denylist(url.Host) {
		http.Error(w, fmt.Sprintf("url [%s] is forbidden", target), http.StatusForbidden)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	r.URL.Host = url.Host
	r.URL.Scheme = url.Scheme
	r.URL.Path = path
	r.Header.Set("X-Forwarded-Host", r.Header.Get("Host"))
	r.Host = url.Host
	proxy.ServeHTTP(w, r)
}

func denylist(host string) bool {
	denied := map[string]struct{}{
		"localhost": {},
		"127.0.0.1": {},
		"0.0.0.0":   {},
		"":          {},
	}
	hostWithoutPort := strings.Split(host, ":")[0]
	_, isDenied := denied[hostWithoutPort]

	return isDenied
}

func isAuthenticated(r *http.Request) bool {
	u, ok := request.UserFrom(r.Context())
	if !ok {
		return false
	}
	authed := false
	for _, g := range u.GetGroups() {
		if g == "system:authenticated" {
			authed = true
		}
	}
	return authed
}
