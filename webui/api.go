package webui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	v1 "github.com/hsmade/minecraft-operator/api/v1"
	"github.com/pkg/errors"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
)

type Api struct {
	Client client.Client
	Log    logr.Logger
}

// getServers gets the manifests for all Servers
func (a *Api) getServers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var servers v1.ServerList
	err := a.Client.List(context.Background(), &servers)
	if err != nil {
		err = errors.Wrap(err, "failed to get Servers")
		a.Log.Info("ERROR failed to get Servers", "error", err)
		returnError(err, w)
		return
	}

	//a.Log.Info("got servers", "servers", servers.Items)
	err = json.NewEncoder(w).Encode(servers.Items)
	if err != nil {
		a.Log.Info("ERROR failed to serialize Servers", "error", err)
		returnError(errors.Wrap(err, "failed to serialize Servers"), w)
		return
	}
}

// setServer sets the status for a Server (enabled: true/false)
func (a *Api) setServer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	enabledString, ok := r.URL.Query()["enabled"]
	if !ok || len(enabledString[0]) < 1 {
		err := errors.New("missing enabled parameter")
		a.Log.Info("parsing parameters", "error", err)
		returnError(err, w)
		return
	}

	enabled, err := strconv.ParseBool(enabledString[0])
	if err != nil {
		err := errors.Wrap(err, "parsing enabled parameter to bool")
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	server, err := a.getServerObject(r)
	if err != nil {
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	a.Log.Info("Got request to set server state", "server", server.Name, "enabled", enabled)

	server.Spec.Enabled = enabled
	a.Log.Info("storing server manifest")
	err = a.Client.Update(context.Background(), server)
	if err != nil {
		err := errors.Wrap(err, "storing server manifest")
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(nil)
}

func (a *Api) postServerCommand(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	server, err := a.getServerObject(r)
	if err != nil {
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	commandString, ok := r.URL.Query()["command"]
	if !ok || len(commandString[0]) < 1 {
		err := errors.New("missing command parameter")
		a.Log.Info("parsing parameters", "error", err)
		returnError(err, w)
		return
	}

	a.Log.Info("Got request to post command to server", "server", server.Name, "command", commandString)

	pod, err := a.getPodForServer(server.Name, server.Namespace)
	if err != nil {
		err := errors.Wrap(err, "getting pod")
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	clientSet, err := a.getApiClient()
	if err != nil {
		err := errors.Wrap(err, "creating k8s api client")
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	req := clientSet.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: "minecraft",
		Stdin:     true,
		Stdout:    false,
		Stderr:    false,
		TTY:       true,
	}, scheme.ParameterCodec)

	config, err := rest.InClusterConfig()
	if err != nil {
		err := errors.Wrap(err, "get cluster config")
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())

	var stdin bytes.Buffer

	stdin.WriteString(commandString[0])
// FIXME: solve
	go func() {
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  &stdin,
			Stdout: nil,
			Stderr: nil,
		})
	}()



}

func (a *Api) getServerLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	server, err := a.getServerObject(r)
	if err != nil {
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	a.Log.Info("Got request for server logs", "server", server.Name)

	logs, err := a.getLogsForServer(server)
	if err != nil {
		err := errors.Wrap(err, "getting pod logs")
		a.Log.Info("ERROR", "error", err)
		returnError(err, w)
		return
	}

	w.WriteHeader(200)
	json.NewEncoder(w).Encode(logs.String())
}

func (a *Api) getPodForServer(name, namespace string)(*corev1.Pod, error) {
	selector := labels.NewSelector()
	appLabel, err := labels.NewRequirement("app", selection.Equals, []string{fmt.Sprintf("minecraft-operator-server-%s", name)})
	if err != nil {
		return nil, errors.Wrap(err, "setting up app selector")
	}

	selector = selector.Add(*appLabel)
	listOptions := client.ListOptions{
		Namespace:     namespace,
		LabelSelector: selector,
		Limit:         10,
	}

	var podList corev1.PodList
	err = a.Client.List(context.Background(), &podList, &listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "listing pods")
	}

	if len(podList.Items) != 1 {
		return nil, errors.New(fmt.Sprintf("found %d pods instead of 1", len(podList.Items)))
	}

	return &podList.Items[0], nil
}
func (a *Api) getLogsForServer(server *v1.Server) (*bytes.Buffer, error) {
	a.Log.Info("looking up pod for server", "server", server.Name)

	pod, err := a.getPodForServer(server.Name, server.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "getting pod for server")
	}

	clientSet, err := a.getApiClient()
	if err != nil {
		return nil, errors.Wrap(err, "creating k8s api client")
	}

	podLogOpts := corev1.PodLogOptions{}
	req := clientSet.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "opening logs stream")
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, errors.Wrap(err, "fetching log from reader")
	}
	return buf, nil
}

func (a *Api) getApiClient() (*kubernetes.Clientset, error){
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get cluster config")
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "create client")
	}

	return clientSet, nil
}

func (a *Api) getServerObject(r *http.Request) (*v1.Server, error) {
	serverName, ok := r.URL.Query()["server"]
	if !ok || len(serverName[0]) < 1 {
		err := errors.New("missing server parameter")
		a.Log.Info("ERROR parsing parameters", "error", err)
		return nil, err
	}

	nameSpace, ok := r.URL.Query()["namespace"]
	if !ok || len(serverName[0]) < 1 {
		err := errors.New("missing namespace parameter")
		a.Log.Info("ERROR parsing parameters", "error", err)
		return nil, err
	}

	var server v1.Server
	err := a.Client.Get(context.Background(), types.NamespacedName{
		Name:      serverName[0],
		Namespace: nameSpace[0],
	}, &server)
	if err != nil {
		err := errors.Wrap(err, "retrieving Server object")
		a.Log.Info("ERROR", "error", err)
		return nil, err
	}

	return &server, nil
}
