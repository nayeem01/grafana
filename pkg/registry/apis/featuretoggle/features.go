package featuretoggle

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"

	common "github.com/grafana/grafana/pkg/apis/common/v0alpha1"
	"github.com/grafana/grafana/pkg/apis/featuretoggle/v0alpha1"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/grafana-apiserver/utils"
)

var (
	_ rest.Storage              = (*featuresStorage)(nil)
	_ rest.Scoper               = (*featuresStorage)(nil)
	_ rest.SingularNameProvider = (*featuresStorage)(nil)
	_ rest.Lister               = (*featuresStorage)(nil)
	_ rest.Getter               = (*featuresStorage)(nil)
)

type featuresStorage struct {
	resource       *common.ResourceInfo
	tableConverter rest.TableConvertor
	features       []featuremgmt.FeatureFlag
	startup        int64
}

// NOTE! this does not depend on config or any system state!
// In the future, the existence of features (and their properties) can be defined dynamically
func NewFeaturesStorage(features []featuremgmt.FeatureFlag) *featuresStorage {
	resourceInfo := v0alpha1.FeatureResourceInfo
	return &featuresStorage{
		startup:  time.Now().UnixMilli(),
		resource: &resourceInfo,
		features: features,
		tableConverter: utils.NewTableConverter(
			resourceInfo.GroupResource(),
			[]metav1.TableColumnDefinition{
				{Name: "Name", Type: "string", Format: "name"},
				{Name: "Stage", Type: "string", Format: "string", Description: "Where is the flag in the dev cycle"},
				{Name: "Owner", Type: "string", Format: "string", Description: "Which team owns the feature"},
			},
			func(obj any) ([]interface{}, error) {
				r, ok := obj.(*v0alpha1.Feature)
				if ok {
					return []interface{}{
						r.Name,
						r.Spec.Stage,
						r.Spec.Owner,
					}, nil
				}
				return nil, fmt.Errorf("expected resource or info")
			}),
	}
}

func (s *featuresStorage) New() runtime.Object {
	return s.resource.NewFunc()
}

func (s *featuresStorage) Destroy() {}

func (s *featuresStorage) NamespaceScoped() bool {
	return false
}

func (s *featuresStorage) GetSingularName() string {
	return s.resource.GetSingularName()
}

func (s *featuresStorage) NewList() runtime.Object {
	return s.resource.NewListFunc()
}

func (s *featuresStorage) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return s.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

func (s *featuresStorage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	flags := &v0alpha1.FeatureList{
		ListMeta: metav1.ListMeta{
			ResourceVersion: fmt.Sprintf("%d", s.startup),
		},
	}
	for _, flag := range s.features {
		flags.Items = append(flags.Items, toK8sForm(flag))
	}
	return flags, nil
}

func (s *featuresStorage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	for _, flag := range s.features {
		if name == flag.Name {
			obj := toK8sForm(flag)
			return &obj, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func toK8sForm(flag featuremgmt.FeatureFlag) v0alpha1.Feature {
	return v0alpha1.Feature{
		ObjectMeta: metav1.ObjectMeta{
			Name:              flag.Name,
			CreationTimestamp: metav1.NewTime(flag.Created),
		},
		Spec: v0alpha1.FeatureSpec{
			Description:       flag.Description,
			Stage:             flag.Stage.String(),
			Owner:             string(flag.Owner),
			AllowSelfServe:    flag.AllowSelfServe,
			HideFromAdminPage: flag.HideFromAdminPage,
			HideFromDocs:      flag.HideFromDocs,
			FrontendOnly:      flag.FrontendOnly,
			RequiresDevMode:   flag.RequiresDevMode,
			RequiresRestart:   flag.RequiresRestart,
		},
	}
}
