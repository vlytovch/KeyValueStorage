package controllers

import (
	"context"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	v12 "kube/api/v1"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	keyValueDataName  = "test-keyvalue"
	keyValueNamespace = "default"
	httpAddress       = "10.96.223.195"
	pairsAddress      = "http://10.96.223.195:8181/pairs"
)

var (
	coreClient          client.Client
	ctx                 = context.TODO()
	initialKeyValueData v12.KeyValueData
	reconciler          *KeyValueDataReconciler
	namespacedName      = types.NamespacedName{Name: keyValueDataName, Namespace: keyValueNamespace}
)

var _ = Describe("KeyValueData controller", func() {

	Context("Succeeded reconcilation flow", func() {

		BeforeEach(func() {
			coreClient = fake.NewClientBuilder().WithScheme(keyValueScheme).Build()
			initialKeyValueData = getKeyValueDataObject()
			reconciler = &KeyValueDataReconciler{
				Client:     coreClient,
				HttpClient: &http.Client{},
				Scheme:     keyValueScheme,
			}
			Expect(coreClient.Create(context.TODO(), getServiceObject())).To(Succeed())
			Expect(coreClient.Create(context.TODO(), &initialKeyValueData)).To(Succeed())
			httpmock.Reset()
		})

		It("should successfully reconcile created CRD, make server requests(GET, PUT) and update status", func() {
			httpmock.RegisterResponder(
				"PUT",
				pairsAddress,
				httpmock.NewStringResponder(201, ""),
			)
			httpmock.RegisterResponder(
				"GET",
				pairsAddress+"/1key1",
				httpmock.NewStringResponder(200, "{\"1key1\": \"1value1\"}"),
			)
			httpmock.RegisterResponder(
				"GET",
				pairsAddress+"/2key2",
				httpmock.NewStringResponder(200, "{\"2key2\": \"2value2\"}"),
			)

			var reconcileResult, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(reconcileResult).To(Equal(ctrl.Result{}))

			callsCount := httpmock.GetCallCountInfo()
			var keyValueData v12.KeyValueData
			coreClient.Get(ctx, types.NamespacedName{
				Name:      keyValueDataName,
				Namespace: keyValueNamespace,
			}, &keyValueData)
			Expect(len(callsCount)).To(Equal(3))
			Expect(callsCount["PUT "+pairsAddress]).To(Equal(2))
			Expect(callsCount["GET "+pairsAddress+"/1key1"]).To(Equal(1))
			Expect(callsCount["GET "+pairsAddress+"/2key2"]).To(Equal(1))
			Expect(keyValueData.GetFinalizers()).To(ContainElement("teamdev.com.keyvaluedata/finalizer"))
			assertCondition(keyValueData.Status.Conditions, "", "", v1.ConditionTrue, v12.KeyValueDataAdded)
			Expect(keyValueData.Status.KeysInStorage).To(ContainElements("1key1", "2key2"))
		})

		It("should successfully reconcile on update, make server requests(GET, PUT, DELETE), update statuses", func() {
			var keyValueData v12.KeyValueData
			Expect(coreClient.Get(context.TODO(), namespacedName, &keyValueData)).To(Succeed())
			keyValueData.Status.KeysInStorage = []string{"1key1", "2key2", "3key3"}
			Expect(coreClient.Update(context.TODO(), &keyValueData)).To(Succeed())
			httpmock.RegisterResponder(
				"DELETE",
				pairsAddress+"/3key3",
				httpmock.NewStringResponder(200, ""),
			)
			httpmock.RegisterResponder(
				"PUT",
				pairsAddress,
				httpmock.NewStringResponder(200, ""),
			)
			httpmock.RegisterResponder(
				"GET",
				pairsAddress+"/1key1",
				httpmock.NewStringResponder(200, "{\"1key1\": \"1value1\"}"),
			)
			httpmock.RegisterResponder(
				"GET",
				pairsAddress+"/2key2",
				httpmock.NewStringResponder(200, "{\"2key2\": \"2value2\"}"),
			)

			var reconcileResult, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(reconcileResult).To(Equal(ctrl.Result{}))

			coreClient.Get(ctx, namespacedName, &keyValueData)
			callsCount := httpmock.GetCallCountInfo()
			Expect(len(callsCount)).To(Equal(4))
			Expect(callsCount["PUT "+pairsAddress]).To(Equal(2))
			Expect(callsCount["GET "+pairsAddress+"/1key1"]).To(Equal(1))
			Expect(callsCount["GET "+pairsAddress+"/2key2"]).To(Equal(1))
			Expect(callsCount["DELETE "+pairsAddress+"/3key3"]).To(Equal(1))
			Expect(keyValueData.GetFinalizers()).To(ContainElement("teamdev.com.keyvaluedata/finalizer"))
			assertCondition(keyValueData.Status.Conditions, "", "", v1.ConditionTrue, v12.KeyValueDataAdded)
			Expect(keyValueData.Status.KeysInStorage).To(ContainElements("2key2"))
		})

		It("should successfully reconcile on create, make server requests(GET, PUT) and write failure reasons to status conditions", func() {
			httpmock.RegisterResponder(
				"PUT",
				pairsAddress,
				httpmock.NewStringResponder(400, "Validation error from server!"),
			)
			httpmock.RegisterResponder(
				"GET",
				pairsAddress+"/2key2",
				httpmock.NewStringResponder(404, "Not found"),
			)
			httpmock.RegisterResponder(
				"GET",
				pairsAddress+"/1key1",
				httpmock.NewStringResponder(404, "Not found"),
			)

			var reconcileResult, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(reconcileResult).To(Equal(ctrl.Result{}))

			var keyValueData v12.KeyValueData
			Expect(coreClient.Get(ctx, namespacedName, &keyValueData)).To(Succeed())
			callsCount := httpmock.GetCallCountInfo()
			Expect(len(callsCount)).To(Equal(3))
			Expect(callsCount["PUT "+pairsAddress]).To(Equal(2))
			Expect(callsCount["GET "+pairsAddress+"/2key2"]).To(Equal(1))
			Expect(callsCount["GET "+pairsAddress+"/1key1"]).To(Equal(1))
			Expect(keyValueData.GetFinalizers()).To(ContainElement("teamdev.com.keyvaluedata/finalizer"))
			assertCondition(keyValueData.Status.Conditions, "BadServerResponse", "[{\"key\":\"1key1\",\"message\":\"Validation error from server!\"},{\"key\":\"2key2\",\"message\":\"Validation error from server!\"}]", v1.ConditionFalse, v12.KeyValueDataAdded)
			Expect(keyValueData.Status.KeysInStorage).To(BeEmpty())
		})

		It("should successfully reconcile on create, make server requests(DELETE), remove finalizer and delete CRD", func() {
			var keyValueData v12.KeyValueData
			Expect(coreClient.Get(context.TODO(), namespacedName, &keyValueData)).To(Succeed())
			keyValueData.SetFinalizers([]string{"teamdev.com.keyvaluedata/finalizer"})
			keyValueData.Status.KeysInStorage = []string{"1key1", "2key2"}
			keyValueData.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})
			coreClient.Update(context.TODO(), &keyValueData)
			httpmock.RegisterResponder(
				"DELETE",
				pairsAddress+"/1key1",
				httpmock.NewStringResponder(200, ""),
			)
			httpmock.RegisterResponder(
				"DELETE",
				pairsAddress+"/2key2",
				httpmock.NewStringResponder(200, ""),
			)

			var asd, _ = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(asd).To(Equal(ctrl.Result{}))

			Expect(coreClient.Get(ctx, namespacedName, &keyValueData)).To(HaveOccurred())
			callsCount := httpmock.GetCallCountInfo()
			Expect(len(callsCount)).To(Equal(2))
			Expect(callsCount["DELETE "+pairsAddress+"/1key1"]).To(Equal(1))
			Expect(callsCount["DELETE "+pairsAddress+"/2key2"]).To(Equal(1))
		})
	})

	Context("Failed reconcilation flow", func() {

		BeforeEach(func() {
			coreClient = fake.NewClientBuilder().WithScheme(keyValueScheme).Build()
			initialKeyValueData = getKeyValueDataObject()
			reconciler = &KeyValueDataReconciler{
				Client:     coreClient,
				HttpClient: &http.Client{},
				Scheme:     keyValueScheme,
			}
			Expect(coreClient.Create(context.TODO(), getServiceObject())).To(Succeed())
			Expect(coreClient.Create(context.TODO(), &initialKeyValueData)).To(Succeed())
			httpmock.Reset()
		})

		It("should result in failed reconcilation when PUT requests to server throw an error", func() {
			var _, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(Not(BeNil()))
		})

		It("should result in failed reconcilation when DELETE requests to server on CRD update throw an error", func() {
			var keyValueData v12.KeyValueData
			Expect(coreClient.Get(context.TODO(), namespacedName, &keyValueData)).To(Succeed())
			keyValueData.Status.KeysInStorage = []string{"1key1", "2key2", "3key3"}
			Expect(coreClient.Update(context.TODO(), &keyValueData)).To(Succeed())

			var _, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())
		})

		It("should result in failed reconcilation when DELETE requests to server on CRD delete throw an error", func() {
			var keyValueData v12.KeyValueData
			Expect(coreClient.Get(context.TODO(), namespacedName, &keyValueData)).To(Succeed())
			keyValueData.SetFinalizers([]string{"teamdev.com.keyvaluedata/finalizer"})
			keyValueData.Status.KeysInStorage = []string{"1key1", "2key2", "3key3"}
			keyValueData.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})
			Expect(coreClient.Update(context.TODO(), &keyValueData)).To(Succeed())

			var _, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())
		})

	})
})

func assertCondition(conditions v12.KeyValueDataConditions, reason, message string, status v1.ConditionStatus, conditionType v12.KeyValueDataConditionType) {
	Expect(conditions).Should(Not(BeEmpty()))
	Expect(conditions[0].Status).Should(Equal(status))
	Expect(conditions[0].Type).Should(Equal(conditionType))
	Expect(conditions[0].Reason).Should(Equal(reason))
	Expect(conditions[0].Message).Should(Equal(message))
	Expect(conditions[0].LastUpdateTime.Time).Should(Not(BeNil()))
}

func getServiceObject() *v1.Service {
	var policy = v1.ServiceInternalTrafficPolicyCluster
	return &v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storage",
			Namespace: "default",
		},
		Spec: v1.ServiceSpec{
			ClusterIP:             httpAddress,
			ClusterIPs:            []string{httpAddress},
			InternalTrafficPolicy: &policy,
			Ports:                 []v1.ServicePort{{TargetPort: intstr.IntOrString{IntVal: 8181}, Port: 8181}},
			Type:                  v1.ServiceTypeClusterIP,
		},
	}
}

func getKeyValueDataObject() v12.KeyValueData {
	return v12.KeyValueData{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "teamdev.com/v1",
			Kind:       "KeyValueData",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      keyValueDataName,
			Namespace: keyValueNamespace,
		},
		Spec: v12.KeyValueDataSpec{
			Data: map[string]string{
				"1key1": "1value1",
				"2key2": "2value2",
			},
		},
	}
}
