package v1

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	keyValueDataName = "keyvalue-name"
	namespace        = "default"
)

var namespacedName = types.NamespacedName{Name: keyValueDataName, Namespace: namespace}

var _ = Describe("KeyValueData Test", func() {

	var keyValueData KeyValueData

	BeforeEach(func() {
		k8sClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		keyValueData = getKeyValueDataObject()
		Expect(k8sClient.Create(context.TODO(), &keyValueData)).To(Succeed())
	})

	Context("Validate", func() {

		It("create validation should not fail when all pairs in created keyvaluepair are unique among all existing CRDs", func() {
			keyValueDataToValidate := getKeyValueDataObject()
			keyValueDataToValidate.Name = "another-keyvalue-name"
			keyValueDataToValidate.Spec.Data = map[string]string{"unique": "uniqueValue"}
			err := keyValueDataToValidate.ValidateCreate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("create validation should fail when keyvaluedata with same pair already exists", func() {
			keyValueDataToValidate := getKeyValueDataObject()
			keyValueDataToValidate.Name = "another-keyvalue-name"
			err := keyValueDataToValidate.ValidateCreate()
			Expect(err).To(MatchError("KeyValueData resource containing \"1key1\" already exists [\"keyvalue-name\"]"))
		})

		It("update validation should not fail while comparing existing and updated keyvaluedata with equal names", func() {
			Expect(k8sClient.Get(context.TODO(), namespacedName, &keyValueData)).To(Succeed())
			err := keyValueData.ValidateUpdate(nil)
			Expect(err).NotTo(HaveOccurred())
		})

		It("update validation should fail if some other existing keyvaluedata contains pair with provided in updated CRD key", func() {
			Expect(k8sClient.Get(context.TODO(), namespacedName, &keyValueData)).To(Succeed())
			newKeyValueData := getKeyValueDataObject()
			newKeyValueData.Name = "another-name"
			Expect(k8sClient.Create(context.TODO(), &newKeyValueData)).To(Succeed())
			err := keyValueData.ValidateUpdate(nil)
			Expect(err).To(MatchError("KeyValueData resource containing \"1key1\" already exists [\"another-name\"]"))
		})

		It("delete validation should not do anything", func() {
			Expect(k8sClient.Get(context.TODO(), namespacedName, &keyValueData)).To(Succeed())
			err := keyValueData.ValidateDelete()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func getKeyValueDataObject() KeyValueData {
	return KeyValueData{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "teamdev.com/v1",
			Kind:       "KeyValueData",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      keyValueDataName,
			Namespace: namespace,
		},
		Spec: KeyValueDataSpec{
			Data: map[string]string{
				"1key1": "1value1",
				"2key2": "2value2",
			},
		},
	}
}
