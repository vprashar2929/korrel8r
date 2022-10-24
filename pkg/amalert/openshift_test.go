package amalert

import (
	"context"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var ctx = context.Background()

func TestAlertManagerMain(t *testing.T) {
	c := fake.NewClientBuilder().Build()
	r := routev1.Route{
		ObjectMeta: v1.ObjectMeta{Name: AlertmanagerMain, Namespace: MonitoringNS},
		Spec:       routev1.RouteSpec{Host: "example.test"},
	}
	assert.NoError(t, c.Create(ctx, &r))
	host, err := OpenshiftManagerHost(c)
	assert.NoError(t, err)
	assert.Equal(t, "example.test", host)
}
