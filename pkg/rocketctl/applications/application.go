package applications

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hex-techs/rocket/pkg/models/application"
	"github.com/hex-techs/rocket/pkg/rocketctl"
	"github.com/hex-techs/rocket/pkg/utils/display"
	"github.com/hex-techs/rocket/pkg/utils/web"
	"github.com/jedib0t/go-pretty/v6/table"
)

func List(filename string) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	var appItems []application.Application
	if err := cli.List(&web.Request{Page: 1, Limit: -1}, &appItems); err != nil {
		return err
	}
	d := display.NewTerminalDisplay("", 0)
	header := table.Row{"name", "namespace", "project", "cloud", "env", "region", "cluster", "create"}
	var content [][]interface{}
	for _, c := range appItems {
		content = append(content, []interface{}{c.Name, c.Namespace, c.Project, c.CloudArea, c.Environment,
			c.Regions, c.Cluster, c.CreatedAt.Local().Format(time.DateTime)})
	}
	d.Table(header, content...)
	return nil
}

func Get(filename, name string) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	var app application.Application
	if err := cli.Get(name, nil, &app); err != nil {
		return err
	}
	b, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

// func Approved(filename string, name string) error {
// 	cli, err := initClient(filename)
// 	if err != nil {
// 		return err
// 	}
// 	var clusterItem cluster.Cluster
// 	if err := cli.Get(name, nil, &clusterItem); err != nil {
// 		return err
// 	}
// 	if clusterItem.State == cluster.Approve {
// 		return fmt.Errorf("the cluster '%s' has been in state of approved", name)
// 	}
// 	d := display.NewTerminalDisplay(fmt.Sprintf("The cluster '%s' is in the state of '%s', Are you sure to approve?",
// 		clusterItem.Name, clusterItem.State), 0)
// 	if d.Confirm() {
// 		clusterItem.State = cluster.Approve
// 		return cli.Update(name, &clusterItem)
// 	}
// 	return nil
// }

func Create(filename string, app *application.Application) error {
	cli, err := rocketctl.InitClient(filename)
	if err != nil {
		return err
	}
	return cli.Create(app)
}
