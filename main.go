package main

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"

	AppsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	StorageV1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type OpenebsPvcStatus struct {
	isDangling bool
	labels     map[string]string
}

func main() {

	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("error %s, getting inclusterconfig", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		// handle error
		fmt.Printf("error %s, creating clientset\n", err.Error())
	}

	ctx := context.Background()

	// To Do - Take selector env var to handle sts deletion
	// To Do - use the labels selectors only for deletion use cases
	// To Do - any pvc that maches the gives selectors and is not mounted is liable for deletion
	// To Do - Ask the developer to put a pre-determine selector on every statefulset whose dangling pvcs are meant to be deleted
	// To Do - Selector goes with delete and scale down and existing sts scale down
	// To Do - Input - Provisioner Names Environment Variable, Storage class annotation to enable dandling pVC delete, Storage Class Parameter with Label Selector - would be provided through storage class parameters

	openEbsStorageClasses := getProspectiveStorageClasses(clientset, ctx)
	if len(openEbsStorageClasses) == 0 {
		panic("No Valid Storage Classes Found")
	}

	openebsPVCsStatus := make(map[string]OpenebsPvcStatus)

	// To Ask - A PVC might have the right storage class and annotation but is could be dangling only if it was created dynamically
	openebsPvcs := getOpenEbsPVCs(clientset, ctx, openEbsStorageClasses)

	allStatefulsets := getAllStatefulSets(clientset, ctx)

	prospectivePvcs := getProspectivePVCs(clientset, ctx, openebsPvcs, allStatefulsets)

	for _, openebsPvc := range prospectivePvcs {
		openebsPVCsStatus[openebsPvc.ObjectMeta.Name] = OpenebsPvcStatus{isDangling: true, labels: openebsPvc.ObjectMeta.Labels}
	}

	for _, statefulset := range allStatefulsets {
		labels := statefulset.Spec.Selector.MatchLabels
		labelsKeys := reflect.ValueOf(labels).MapKeys()
		key := labelsKeys[0].Interface().(string)
		selector := key + "=" + labels[key]

		pods, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{LabelSelector: selector})

		if err != nil {
			fmt.Printf("error %s, getting PVCs\n", err.Error())
		}

		for _, pod := range pods.Items {
			podVolumes := pod.Spec.Volumes

			for _, volume := range podVolumes {
				if volume.PersistentVolumeClaim != nil {
					entry, found := openebsPVCsStatus[volume.PersistentVolumeClaim.ClaimName]
					if found {
						entry.isDangling = false
						openebsPVCsStatus[volume.PersistentVolumeClaim.ClaimName] = entry
					}
				}
			}
		}
	}

	for pvc, status := range openebsPVCsStatus {
		if status.isDangling {
			fmt.Println(pvc + " is dangling!")
			/*
				err := clientset.CoreV1().PersistentVolumeClaims("default").Delete(ctx, pvc, metav1.DeleteOptions{})

				if err == nil {
					fmt.Printf("Dangling PVC %s deleted successfully\n", pvc)
				}
			*/
		}
	}
}

// Gets dynamically created PVCs from list of OpenEBS PVCs with annotation provided
func getProspectivePVCs(clientset *kubernetes.Clientset, ctx context.Context, pvcs []v1.PersistentVolumeClaim, statefulsets []AppsV1.StatefulSet) []v1.PersistentVolumeClaim {
	// Dynamic PVCs have the same label as their stateful set
	var dynamicPvcs []v1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		// To Ask - Do PVCs take all the labels of their Stateful Set or only a few?
		// To Ask - What if the entire stateful set is deleted?, we would have PVCs with labels but how would we know it was greated dynamically
		for _, statefulset := range statefulsets {
			if reflect.DeepEqual(statefulset.Spec.Selector.MatchLabels, pvc.Labels) {
				dynamicPvcs = append(dynamicPvcs, pvc)
			}
		}
	}
	return dynamicPvcs
}

func getAllStatefulSets(clientset *kubernetes.Clientset, ctx context.Context) []AppsV1.StatefulSet {
	allStatefulsets, errAllSts := clientset.AppsV1().StatefulSets("default").List(ctx, metav1.ListOptions{})
	if errAllSts != nil {
		fmt.Printf("error %s, getting PVCs\n", errAllSts.Error())
	}
	return allStatefulsets.Items
}

func getOpenEbsPVCs(clientset *kubernetes.Clientset, ctx context.Context, storageclasses []StorageV1.StorageClass) []v1.PersistentVolumeClaim {
	allPvcs, errPVC := clientset.CoreV1().PersistentVolumeClaims("default").List(ctx, metav1.ListOptions{})
	if errPVC != nil {
		fmt.Printf("error %s, getting PVCs\n", errPVC.Error())
	}

	var openebsPvcs []v1.PersistentVolumeClaim

	for _, pvc := range allPvcs.Items {
		pvcStorageClassName := *pvc.Spec.StorageClassName
		for _, openEbsStorageClass := range storageclasses {
			if pvcStorageClassName == openEbsStorageClass.Name {
				openebsPvcs = append(openebsPvcs, pvc)
			}

		}
	}
	return openebsPvcs
}

func getProspectiveStorageClasses(clientset *kubernetes.Clientset, ctx context.Context) []StorageV1.StorageClass {

	provisionersEnvVar, exists := os.LookupEnv("PROVISIONERS")
	if !exists {
		panic("Required Environment Variable PROVISIONERS not found")
	}
	provisioners := strings.Split(provisionersEnvVar, ",")

	allSc, errSc := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if errSc != nil {
		fmt.Printf("error %s, getting Storage Classes\n", errSc.Error())
	}

	var openEbsStorageClasses []StorageV1.StorageClass

	for _, storageclass := range allSc.Items {
		for _, openEbsProvisioner := range provisioners {
			if storageclass.Provisioner == openEbsProvisioner && storageclass.Annotations["openebs.io/delete-dangling-pvc"] != "true" {
				openEbsStorageClasses = append(openEbsStorageClasses, storageclass)
			}
		}
	}
	return openEbsStorageClasses
}
