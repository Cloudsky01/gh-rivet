package tui

import (
	"fmt"

	"github.com/Cloudsky01/gh-rivet/internal/config"
	"github.com/Cloudsky01/gh-rivet/internal/tui/components"
)

type navItemData struct {
	isGroup      bool
	group        *config.Group
	workflowName string
	isPinned     bool
}

func (a *App) refreshNavList() {
	items := a.buildNavItems()
	a.navList.SetItems(items)

	if len(a.groupPath) == 0 {
		a.navList.SetTitle("📁 Groups")
	} else {
		current := a.groupPath[len(a.groupPath)-1]
		a.navList.SetTitle("📁 " + current.Name)
	}
}

func (a *App) buildNavItems() []components.ListItem {
	var items []components.ListItem

	if len(a.groupPath) == 0 {
		items = a.buildRootGroupItems()
	} else {
		items = a.buildCurrentGroupItems()
	}

	return items
}

func (a *App) buildRootGroupItems() []components.ListItem {
	var items []components.ListItem
	for i := range a.config.Groups {
		group := &a.config.Groups[i]
		items = append(items, a.createGroupListItem(group))
	}
	return items
}

func (a *App) buildCurrentGroupItems() []components.ListItem {
	var items []components.ListItem
	currentGroup := a.groupPath[len(a.groupPath)-1]

	workflows := a.collectWorkflows(currentGroup)
	workflowDefs := a.buildWorkflowDefsMap(currentGroup)
	pinnedWorkflows, unpinnedWorkflows := a.separatePinnedWorkflows(currentGroup, workflows)

	items = append(items, a.createWorkflowItems(pinnedWorkflows, workflowDefs, true)...)
	items = append(items, a.createWorkflowItems(unpinnedWorkflows, workflowDefs, false)...)

	for i := range currentGroup.Groups {
		group := &currentGroup.Groups[i]
		items = append(items, a.createGroupListItem(group))
	}

	return items
}

func (a *App) collectWorkflows(group *config.Group) []string {
	workflows := make([]string, 0)
	workflows = append(workflows, group.Workflows...)

	for i := range group.WorkflowDefs {
		wf := &group.WorkflowDefs[i]
		if !contains(workflows, wf.File) {
			workflows = append(workflows, wf.File)
		}
	}

	return workflows
}

func (a *App) buildWorkflowDefsMap(group *config.Group) map[string]*config.Workflow {
	workflowDefs := make(map[string]*config.Workflow)
	for i := range group.WorkflowDefs {
		wf := &group.WorkflowDefs[i]
		workflowDefs[wf.File] = wf
	}
	return workflowDefs
}

func (a *App) separatePinnedWorkflows(group *config.Group, workflows []string) ([]string, []string) {
	var pinnedWorkflows []string
	var unpinnedWorkflows []string

	for _, wf := range workflows {
		if group.IsPinned(wf) {
			pinnedWorkflows = append(pinnedWorkflows, wf)
		} else {
			unpinnedWorkflows = append(unpinnedWorkflows, wf)
		}
	}

	return pinnedWorkflows, unpinnedWorkflows
}

func (a *App) createWorkflowItems(workflows []string, workflowDefs map[string]*config.Workflow, isPinned bool) []components.ListItem {
	items := make([]components.ListItem, 0, len(workflows))

	for _, wf := range workflows {
		displayName := wf
		if wfDef, ok := workflowDefs[wf]; ok && wfDef.Name != "" {
			displayName = wfDef.Name
		}

		icon := a.theme.Icons.Workflow
		if isPinned {
			icon = a.theme.Icons.Pin
		}

		items = append(items, components.ListItem{
			ID:          wf,
			Title:       displayName,
			Description: wf,
			Icon:        icon,
			Data: &navItemData{
				isGroup:      false,
				workflowName: wf,
				isPinned:     isPinned,
			},
		})
	}

	return items
}

func (a *App) createGroupListItem(group *config.Group) components.ListItem {
	return components.ListItem{
		ID:          group.ID,
		Title:       group.Name,
		Description: fmt.Sprintf("%d workflows", a.countWorkflows(group)),
		Icon:        a.theme.Icons.Folder,
		Data: &navItemData{
			isGroup: true,
			group:   group,
		},
	}
}

func (a *App) countWorkflows(group *config.Group) int {
	count := len(group.Workflows) + len(group.WorkflowDefs)
	for i := range group.Groups {
		count += a.countWorkflows(&group.Groups[i])
	}
	return count
}

func (a *App) refreshPinnedList() {
	pinnedWorkflows := a.config.GetAllPinnedWorkflows()
	items := make([]components.PinnedItem, len(pinnedWorkflows))

	for i, pw := range pinnedWorkflows {
		items[i] = components.PinnedItem{
			WorkflowName: pw.WorkflowName,
			GroupName:    pw.Group.Name,
			GroupID:      pw.Group.ID,
			Data:         pw.Group,
		}
	}

	a.sidebar.SetItems(items)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
