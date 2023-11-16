package cluster

import (
	"context"
	"encoding/json"
	"time"

	"github.com/fize/go-ext/log"
	"github.com/google/go-cmp/cmp"
	"github.com/hex-techs/rocket/pkg/models/cluster"
	"github.com/hex-techs/rocket/pkg/models/revisionmanager"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"gorm.io/gorm"
)

var (
	interval = 270 * time.Second
	timeout  = 300 * time.Second
)

func (cc *ClusterController) heartbeatChecker() {
	var clss []cluster.Cluster
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		ctx := context.Background()
		total, err := cc.store.List(ctx, -1, 1, "", &clss)
		if err != nil {
			log.Errorf("list cluster error when check heartbeat: %v", err)
			continue
		}
		if total == 0 {
			log.Debug("no cluster found to check heartbeat")
			continue
		}
		for _, cls := range clss {
			if !cls.Agent {
				log.Debugw("cluster is not in agent mode, ingore heartbeat check", "cluster", cls.Name, "id", cls.ID)
				continue
			}
			if !cls.LastKeepAliveTime.IsZero() {
				expire := cls.LastKeepAliveTime.Add(timeout)
				if time.Now().After(expire) {
					// overtime
					if cls.State == cluster.Approve {
						cls.State = cluster.Offline
					}
				} else {
					// recover from offline
					if cls.State == cluster.Reject || cls.State == cluster.Pending {
						continue
					}
					if cls.State == cluster.Offline {
						cls.State = cluster.Approve
					}
				}
			}
			if cls.State == "" {
				log.Warnw("cluster state is empty", "cluster", cls.Name, "id", cls.ID)
				continue
			}
			if err := cc.store.Update(ctx, cls.ID, "", &cluster.Cluster{}, &cls); err != nil {
				log.Errorf("update cluster '%s(id:%d)' error when check heartbeat: %v", cls.Name, cls.ID, err)
			}
		}
	}
}

func (cc *ClusterController) storeRevisionManager(o, n *cluster.Cluster) {
	otmp, err := json.Marshal(o)
	if err != nil {
		log.Errorw("failed to marshal cluster", "error", err, "cluster", o)
		return
	}
	if o.ID != n.ID {
		log.Errorw("cluster id must be same", "old", o.ID, "new", n.ID)
		return
	}
	o.Description = ""
	o.DeletedAt = gorm.DeletedAt{}
	o.UpdatedAt = time.Time{}
	o.LastKeepAliveTime = time.Time{}
	o.ReadyNodeCount = 0
	o.UnhealthNodeCount = 0
	o.AllocatableCPU = ""
	o.AllocatableMemory = ""
	o.CapacityCPU = ""
	o.CapacityMemory = ""
	o.State = ""
	n.Description = ""
	n.DeletedAt = gorm.DeletedAt{}
	n.UpdatedAt = time.Time{}
	n.LastKeepAliveTime = time.Time{}
	n.ReadyNodeCount = 0
	n.UnhealthNodeCount = 0
	n.AllocatableCPU = ""
	n.AllocatableMemory = ""
	n.CapacityCPU = ""
	n.CapacityMemory = ""
	n.State = ""
	if !cmp.Equal(o, n) {
		ver := revisionmanager.RevisionManager{
			ResourceKind: constant.ClusterRevisionKind,
			ResourceName: o.Name,
			ResourceID:   o.ID,
			Version:      tools.GenerateVersionWithTime(),
			Content:      string(otmp),
		}
		if err := cc.store.Create(context.Background(), &ver); err != nil {
			log.Errorw("failed to create revision manager", "error", err, "cluster", o)
		}
	}
}

func (cc *ClusterController) deleteRevisionManager(o *cluster.Cluster) error {
	// delete revision manager when cluster is deleted
	if err := cc.store.Client().(*gorm.DB).
		Where("resource_kind = ? and resource_id = ?", constant.ClusterRevisionKind, o.ID).
		Delete(&revisionmanager.RevisionManager{}).Error; err != nil {
		return err
	}
	return nil
}
