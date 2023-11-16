package application

import (
	"context"
	"encoding/json"

	"github.com/fize/go-ext/log"
	"github.com/google/go-cmp/cmp"
	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/models/revisionmanager"
	"github.com/hex-techs/rocket/pkg/utils/constant"
	"github.com/hex-techs/rocket/pkg/utils/tools"
	"gorm.io/gorm"
)

func (ac *ApplicationController) storeRevisionManager(o, n *application.Application) {
	otmp, err := json.Marshal(o)
	if err != nil {
		log.Errorw("failed to marshal application", "error", err, "application", o)
		return
	}
	if o.ID != n.ID {
		log.Errorw("application id must be same", "old", o.ID, "new", n.ID)
		return
	}
	if !cmp.Equal(o.Template, n.Template) {
		ver := revisionmanager.RevisionManager{
			ResourceKind: constant.ApplicationRevisionKind,
			ResourceName: o.Name,
			Version:      tools.GenerateVersionWithTime(),
			Content:      string(otmp),
		}
		if err := ac.store.Create(context.Background(), &ver); err != nil {
			log.Errorw("failed to create revision manager", "error", err, "application", o)
		}
	}
}

func (ac *ApplicationController) deleteRevisionManager(o *application.Application) error {
	// delete revision manager when application is deleted
	if err := ac.store.Client().(*gorm.DB).
		Where("resource_kind = ? and resource_id = ?", constant.ApplicationRevisionKind, o.ID).
		Delete(&revisionmanager.RevisionManager{}).Error; err != nil {
		return err
	}
	return nil
}
