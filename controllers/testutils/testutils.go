// Package testutils provides utilities for testing ApplicationGroup Controller.
package testutils

import (
	"math/rand"

	"github.com/Azure/Orkestra/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AddApplication(appGroup v1alpha1.ApplicationGroup, app v1alpha1.Application) v1alpha1.ApplicationGroup {
	appGroup.Spec.Applications = append(appGroup.Spec.Applications, app)
	return appGroup
}

func DefaultAppGroup(groupName, groupNamespace, targetNamespace string) *v1alpha1.ApplicationGroup {
	g := &v1alpha1.ApplicationGroup{
		ObjectMeta: v1.ObjectMeta{
			Name:      groupName,
			Namespace: groupNamespace,
		},
	}
	g.Spec.Applications = make([]v1alpha1.Application, 0)
	g.Spec.Applications = append(g.Spec.Applications, BookinfoApplication(targetNamespace), AmbassadorApplication(targetNamespace))
	return g
}

func SmallAppGroup(groupName, groupNamespace, targetNamespace string) *v1alpha1.ApplicationGroup {
	g := &v1alpha1.ApplicationGroup{
		ObjectMeta: v1.ObjectMeta{
			Name:      groupName,
			Namespace: groupNamespace,
		},
	}
	g.Spec.Applications = make([]v1alpha1.Application, 0)
	g.Spec.Applications = append(g.Spec.Applications, PodinfoApplication(targetNamespace))
	return g
}

func CreateAppGroupName(name string) string {
	return name + "-" + GetRandomStringRunes(10)
}

func GetRandomStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func BoolToBoolPtr(in bool) *bool {
	return &in
}
