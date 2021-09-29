package dns

import (
	"github.com/kubesphere/kubekey/pkg/core/action"
	"github.com/kubesphere/kubekey/pkg/core/modules"
	"github.com/kubesphere/kubekey/pkg/core/prepare"
	"github.com/kubesphere/kubekey/pkg/core/util"
	"github.com/kubesphere/kubekey/pkg/pipelines/common"
	"github.com/kubesphere/kubekey/pkg/pipelines/images"
	"github.com/kubesphere/kubekey/pkg/pipelines/plugins/dns/templates"
	"path/filepath"
)

type ClusterDNSModule struct {
	common.KubeModule
}

func (c *ClusterDNSModule) Init() {
	c.Name = "ClusterDNSModule"

	generateCoreDNDSvc := &modules.RemoteTask{
		Name:  "GenerateCoreDNSSvc",
		Desc:  "generate coredns service",
		Hosts: c.Runtime.GetHostsByRole(common.Master),
		Prepare: &prepare.PrepareCollection{
			new(common.OnlyFirstMaster),
			&CoreDNSExist{Not: true},
		},
		Action: &action.Template{
			Template: templates.CorednsService,
			Dst:      filepath.Join(common.KubeConfigDir, templates.CorednsService.Name()),
			Data: util.Data{
				"ClusterIP": c.KubeConf.Cluster.CorednsClusterIP(),
			},
		},
		Parallel: true,
	}

	override := &modules.RemoteTask{
		Name:  "OverrideCoreDNSService",
		Desc:  "override coredns service",
		Hosts: c.Runtime.GetHostsByRole(common.Master),
		Prepare: &prepare.PrepareCollection{
			new(common.OnlyFirstMaster),
			&CoreDNSExist{Not: true},
		},
		Action:   new(OverrideCoreDNS),
		Parallel: true,
	}

	generateNodeLocalDNS := &modules.RemoteTask{
		Name:  "GenerateNodeLocalDNS",
		Desc:  "generate nodelocaldns",
		Hosts: c.Runtime.GetHostsByRole(common.Master),
		Prepare: &prepare.PrepareCollection{
			new(common.OnlyFirstMaster),
			new(EnableNodeLocalDNS),
		},
		Action: &action.Template{
			Template: templates.NodeLocalDNSService,
			Dst:      filepath.Join(common.KubeConfigDir, templates.NodeLocalDNSService.Name()),
			Data: util.Data{
				"NodelocaldnsImage": images.GetImage(c.Runtime, c.KubeConf, "k8s-dns-node-cache").ImageName(),
			},
		},
		Parallel: true,
	}

	applyNodeLocalDNS := &modules.RemoteTask{
		Name:  "DeployNodeLocalDNS",
		Desc:  "deploy nodelocaldns",
		Hosts: c.Runtime.GetHostsByRole(common.Master),
		Prepare: &prepare.PrepareCollection{
			new(common.OnlyFirstMaster),
			new(EnableNodeLocalDNS),
		},
		Action:   new(DeployNodeLocalDNS),
		Parallel: true,
		Retry:    5,
	}

	generateNodeLocalDNSConfigMap := &modules.RemoteTask{
		Name:  "GenerateNodeLocalDNSConfigMap",
		Desc:  "generate nodelocaldns configmap",
		Hosts: c.Runtime.GetHostsByRole(common.Master),
		Prepare: &prepare.PrepareCollection{
			new(common.OnlyFirstMaster),
			new(EnableNodeLocalDNS),
			new(NodeLocalDNSConfigMapNotExist),
		},
		Action:   new(GenerateNodeLocalDNSConfigMap),
		Parallel: true,
	}

	applyNodeLocalDNSConfigMap := &modules.RemoteTask{
		Name:  "ApplyNodeLocalDNSConfigMap",
		Desc:  "apply nodelocaldns configmap",
		Hosts: c.Runtime.GetHostsByRole(common.Master),
		Prepare: &prepare.PrepareCollection{
			new(common.OnlyFirstMaster),
			new(EnableNodeLocalDNS),
			new(NodeLocalDNSConfigMapNotExist),
		},
		Action:   new(ApplyNodeLocalDNSConfigMap),
		Parallel: true,
		Retry:    5,
	}

	c.Tasks = []modules.Task{
		generateCoreDNDSvc,
		override,
		generateNodeLocalDNS,
		applyNodeLocalDNS,
		generateNodeLocalDNSConfigMap,
		applyNodeLocalDNSConfigMap,
	}
}
