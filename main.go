package main

import (
	"context"
	"flag"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "/home/vivek/.kube/config", "location to your kubeconfig file")
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		// handle error
		fmt.Printf("erorr %s building config from flags\n", err.Error())
		config, err = rest.InClusterConfig()
		if err != nil {
			fmt.Printf("error %s, getting inclusterconfig", err.Error())
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		// handle error
		fmt.Printf("error %s, creating clientset\n", err.Error())
	}
	ctx := context.Background()

	pods, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		// handle error
		fmt.Printf("error %s, while listing all the pods from default namespace\n", err.Error())
	}
	fmt.Println("Pods from default namespace")
	for _, pod := range pods.Items {
		fmt.Println(pod)
	}

	// pvcs, err := clientset.CoreV1().PersistentVolumeClaims("default").List(ctx, metav1.ListOptions{})
	// for _, pvc := range pvcs.Items {
	// 	pvcdetails, err := clientset.CoreV1().PersistentVolumeClaims("default").Get(ctx, pvc.Name, metav1.GetOptions{})
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	} else {
	// 		fmt.Println(pvcdetails.ManagedFields)
	// 		fmt.Println(pvcdetails.OwnerReferences)
	// 		fmt.Println(pvcdetails.ObjectMeta)
	// 		fmt.Println(pvcdetails.Status)
	// 		pvcdetails.eve
	// 		fmt.Printf("\n")
	// 		fmt.Printf("\n")
	// 		fmt.Printf("\n")
	// 		fmt.Printf("\n")
	// 	}
	// }

	// fmt.Println(pvcs)

	// fmt.Println("Deployments are")
	// deployments, err := clientset.AppsV1().Deployments("default").List(ctx, metav1.ListOptions{})
	// if err != nil {
	// 	fmt.Printf("listing deployments %s\n", err.Error())
	// }
	// for _, d := range deployments.Items {
	// 	fmt.Printf("%s", d.Name)
	// }
}
