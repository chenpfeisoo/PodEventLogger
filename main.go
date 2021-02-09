package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	//	"github.com/bndr/gotabulate"
)

var (
	clientset *kubernetes.Clientset
	err       error
)

const (
	Pulling           = "Pulling"
	Pulled            = "Pulled"
	Started           = "Started"
	Created           = "Created"
	SuccessfulCreate  = "SuccessfulCreate"
	ScalingReplicaSet = "ScalingReplicaSet"
)
type  PullImageInfo struct {
	EventName string `json:"event_name"`
	ImageName  string `json:"image_name"`
	PullimageTime float64 `json:"pullimage_time"`
}
func (c *PullImageInfo) ToJsonString() string {
	b, _ := json.Marshal(c)
	return string(b)
}
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
func createClient() (clientset *kubernetes.Clientset, err error) {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	}
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err = kubernetes.NewForConfig(config)
	return
}
func main() {

	clientset, err = createClient()
	if err != nil {
		klog.Error(err.Error())
	}

	wait.Forever(podlog, 30*time.Second)
}
func podlog() {

	data, err := clientset.RESTClient().Get().AbsPath("api/v1/events").DoRaw()
	if err != nil {
		klog.Error("can't get events: " + err.Error())
	}
	var events v1.EventList
	err = json.Unmarshal(data, &events)
	if err != nil {
		klog.Error("error in parsing json: " + err.Error())
	}
	pullImagetime(events)
	runContainerTime(events)
}
func pullImagetime(events v1.EventList) {
	startmap := pullingStartTime(events)
	endmap := pulledFinTime(events)
	for k, v := range startmap {
		for i, j := range endmap {
			if i == k {
				//找到同一个depoly
				pt:=PullImageInfo{
					EventName: strings.Split(i, ":")[0],
					ImageName: strings.Split(i, ":")[1],
					PullimageTime: j.Sub(v).Seconds(),
				}
				klog.Info(pt.ToJsonString())
			}
		}
	}
}

//开始pull时间
func pullingStartTime(events v1.EventList) map[string]time.Time {
	for _, event := range events.Items {
		if strings.Contains(event.Reason, Pulling) {
			return map[string]time.Time{
				eventName(event.Name)+":"+strings.Trim(imageName(event.Message), "\""): event.CreationTimestamp.Time,
			}
		}
	}
	return nil
}

//pulled完成时间
func pulledFinTime(events v1.EventList) map[string]time.Time {
	for _, event := range events.Items {
		if strings.Contains(event.Reason, Pulled) {
			return map[string]time.Time{
				eventName(event.Name)+":"+strings.Trim(imageName(event.Message), "\""): event.LastTimestamp.Time,
			}
		}
	}
	return nil
}

//处理一下event.Name
func eventName(str string) string {
	v := strings.Split(str, ".")
	return v[0]
}

func imageName(str string) string {
	v := strings.Split(str, " ")
	return v[len(v)-1]
}
func runContainerTime(events v1.EventList) {
	startmap := createContainerTime(events)
	endmap := scalingReplicaSet(events)
	for k, v := range endmap {
		for i, j := range startmap {
			if strings.Contains(k, i) {
				//找到同一个depoly
				klog.Infof("EventName\t%v\trunContainerTime\t%vs", i, j.Sub(v).Seconds())
			}
		}
	}
}

//Created pod时间
func createContainerTime(events v1.EventList) map[string]time.Time {
	for _, event := range events.Items {
		if strings.Contains(event.Reason, Created) {
			return map[string]time.Time{
				eventName(event.Name): event.CreationTimestamp.Time,
			}
		}
	}
	return nil
}

//scalingReplicaSet完成时间
func scalingReplicaSet(events v1.EventList) map[string]time.Time {
	for _, event := range events.Items {
		if strings.Contains(event.Reason, ScalingReplicaSet) {
			return map[string]time.Time{
				eventName(event.Name): event.LastTimestamp.Time,
			}
		}
	}
	return nil
}
