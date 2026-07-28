package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ydcloud-dy/opshub/plugins/app-inventory/model"
)

type TopologyNode struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Status   string                 `json:"status"`
	AppID    uint                   `json:"appId,omitempty"`
	EnvID    uint                   `json:"environmentId,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type TopologyEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Label    string `json:"label"`
	Protocol string `json:"protocol,omitempty"`
	Status   string `json:"status"`
	Kind     string `json:"kind"`
}

type TopologyResponse struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
	Stats map[string]int `json:"stats"`
}

func (s *Service) Topology(ctx context.Context, appID uint) (*TopologyResponse, error) {
	var apps []model.Application
	if appID > 0 {
		var selected model.Application
		if err := s.db.WithContext(ctx).First(&selected, appID).Error; err != nil {
			return nil, err
		}
		apps = append(apps, selected)
	} else if err := s.db.WithContext(ctx).Order("name").Find(&apps).Error; err != nil {
		return nil, err
	}

	var dependencies []model.Dependency
	q := s.db.WithContext(ctx).Where("status <> ?", "disabled")
	if appID > 0 {
		componentIDs := s.db.WithContext(ctx).Model(&model.Component{}).Select("id").Where("application_id = ?", appID)
		resourceIDs := s.db.WithContext(ctx).Model(&model.Resource{}).Select("id").Where("application_id = ?", appID)
		q = q.Where(
			"source_application_id = ? OR target_application_id = ? OR target_component_id IN (?) OR target_resource_id IN (?)",
			appID, appID, componentIDs, resourceIDs,
		)
	}
	if err := q.Order("id").Find(&dependencies).Error; err != nil {
		return nil, err
	}
	appMap := make(map[uint]model.Application, len(apps))
	for _, app := range apps {
		appMap[app.ID] = app
	}
	// A selected application's dependencies may point to applications outside the initial set.
	for _, dep := range dependencies {
		if dep.TargetApplicationID > 0 {
			if _, ok := appMap[dep.TargetApplicationID]; !ok {
				var target model.Application
				if err := s.db.WithContext(ctx).First(&target, dep.TargetApplicationID).Error; err == nil {
					appMap[target.ID] = target
				}
			}
		}
		if appID > 0 && dep.SourceApplicationID > 0 {
			if _, ok := appMap[dep.SourceApplicationID]; !ok {
				var source model.Application
				if err := s.db.WithContext(ctx).First(&source, dep.SourceApplicationID).Error; err == nil {
					appMap[source.ID] = source
				}
			}
		}
	}

	result := &TopologyResponse{Nodes: make([]TopologyNode, 0), Edges: make([]TopologyEdge, 0), Stats: map[string]int{}}
	nodeSeen := map[string]bool{}
	addNode := func(node TopologyNode) {
		if nodeSeen[node.ID] {
			return
		}
		nodeSeen[node.ID] = true
		result.Nodes = append(result.Nodes, node)
	}
	addEdge := func(edge TopologyEdge) {
		result.Edges = append(result.Edges, edge)
	}
	for _, app := range appMap {
		addNode(TopologyNode{ID: appNodeID(app.ID), Label: app.Name, Type: "application", Status: app.HealthStatus, AppID: app.ID, Metadata: applicationTopologyMetadata(app)})
	}

	if appID > 0 {
		selected := appMap[appID]
		if selected.EnvironmentID > 0 {
			var env model.Environment
			if err := s.db.WithContext(ctx).First(&env, selected.EnvironmentID).Error; err != nil {
				return nil, err
			}
			envID := envNodeID(env.ID)
			addNode(TopologyNode{ID: envID, Label: env.Name, Type: "environment", Status: env.Status, AppID: selected.ID, EnvID: env.ID, Metadata: map[string]interface{}{"kind": env.Kind, "region": env.Region}})
			addEdge(TopologyEdge{ID: fmt.Sprintf("edge-app-%d-env-%d", selected.ID, env.ID), Source: appNodeID(selected.ID), Target: envID, Label: "运行环境", Status: env.Status, Kind: "structure"})
		}
		var resources []model.Resource
		if err := s.db.WithContext(ctx).Where("application_id = ?", appID).Order("kind, id").Find(&resources).Error; err != nil {
			return nil, err
		}
		for _, resource := range resources {
			resourceNode := TopologyNode{ID: resourceNodeID(resource.ID), Label: resource.Name, Type: "resource", Status: resource.Status, AppID: resource.ApplicationID, EnvID: resource.EnvironmentID, Metadata: map[string]interface{}{"kind": resource.Kind, "address": resource.Address, "port": resource.Port, "namespace": resource.Namespace, "source": resource.Source}}
			addNode(resourceNode)
			parent := appNodeID(resource.ApplicationID)
			if resource.EnvironmentID > 0 {
				parent = envNodeID(resource.EnvironmentID)
			}
			addEdge(TopologyEdge{ID: fmt.Sprintf("edge-resource-%d", resource.ID), Source: parent, Target: resourceNodeID(resource.ID), Label: resource.Kind, Status: resource.Status, Kind: "structure"})
		}
		var components []model.Component
		if err := s.db.WithContext(ctx).Where("application_id = ?", appID).Order("category, id").Find(&components).Error; err != nil {
			return nil, err
		}
		for _, component := range components {
			addNode(TopologyNode{ID: componentNodeID(component.ID), Label: component.Name, Type: "component", Status: component.Status, AppID: component.ApplicationID, EnvID: component.EnvironmentID, Metadata: map[string]interface{}{"category": component.Category, "type": component.Type, "address": component.Address, "port": component.Port}})
			parent := appNodeID(component.ApplicationID)
			if component.EnvironmentID > 0 {
				parent = envNodeID(component.EnvironmentID)
			}
			addEdge(TopologyEdge{ID: fmt.Sprintf("edge-component-%d", component.ID), Source: parent, Target: componentNodeID(component.ID), Label: component.Category, Status: component.Status, Kind: "structure"})
		}
		var domains []model.Domain
		if err := s.db.WithContext(ctx).Where("application_id = ?", appID).Order("domain").Find(&domains).Error; err != nil {
			return nil, err
		}
		for _, domain := range domains {
			addNode(TopologyNode{ID: domainNodeID(domain.ID), Label: domain.Domain, Type: "domain", Status: domain.Status, AppID: domain.ApplicationID, EnvID: domain.EnvironmentID, Metadata: map[string]interface{}{"protocol": domain.Protocol, "port": domain.Port, "certificateId": domain.CertificateID}})
			parent := appNodeID(domain.ApplicationID)
			if domain.EnvironmentID > 0 {
				parent = envNodeID(domain.EnvironmentID)
			}
			addEdge(TopologyEdge{ID: fmt.Sprintf("edge-domain-%d", domain.ID), Source: domainNodeID(domain.ID), Target: parent, Label: "访问入口", Status: domain.Status, Kind: "structure"})
		}
	}

	for _, dep := range dependencies {
		source := appNodeID(dep.SourceApplicationID)
		target := dependencyTargetID(dep)
		if dep.TargetApplicationID == 0 && dep.TargetComponentID == 0 && dep.TargetResourceID == 0 {
			target = externalNodeID(dep.ID)
			addNode(TopologyNode{ID: target, Label: firstNonEmpty(dep.TargetName, dep.Endpoint, "外部依赖"), Type: "external", Status: dep.Status, Metadata: map[string]interface{}{"endpoint": dep.Endpoint, "port": dep.Port}})
		}
		if dep.TargetComponentID > 0 && !nodeSeen[target] {
			var component model.Component
			if err := s.db.WithContext(ctx).First(&component, dep.TargetComponentID).Error; err == nil {
				addNode(TopologyNode{ID: target, Label: component.Name, Type: "component", Status: component.Status, AppID: component.ApplicationID, EnvID: component.EnvironmentID, Metadata: map[string]interface{}{"category": component.Category, "type": component.Type, "address": component.Address, "port": component.Port}})
			}
		}
		if dep.TargetResourceID > 0 && !nodeSeen[target] {
			var resource model.Resource
			if err := s.db.WithContext(ctx).First(&resource, dep.TargetResourceID).Error; err == nil {
				addNode(TopologyNode{ID: target, Label: resource.Name, Type: "resource", Status: resource.Status, AppID: resource.ApplicationID, EnvID: resource.EnvironmentID, Metadata: map[string]interface{}{"kind": resource.Kind, "address": resource.Address, "port": resource.Port}})
			}
		}
		addEdge(TopologyEdge{ID: fmt.Sprintf("dependency-%d", dep.ID), Source: source, Target: target, Label: firstNonEmpty(dep.RelationType, "调用"), Protocol: dep.Protocol, Status: dep.Status, Kind: "dependency"})
	}
	for _, node := range result.Nodes {
		result.Stats[node.Type]++
	}
	result.Stats["edges"] = len(result.Edges)
	return result, nil
}

func appNodeID(id uint) string       { return fmt.Sprintf("app:%d", id) }
func envNodeID(id uint) string       { return fmt.Sprintf("env:%d", id) }
func resourceNodeID(id uint) string  { return fmt.Sprintf("resource:%d", id) }
func componentNodeID(id uint) string { return fmt.Sprintf("component:%d", id) }
func domainNodeID(id uint) string    { return fmt.Sprintf("domain:%d", id) }
func externalNodeID(id uint) string  { return fmt.Sprintf("external:%d", id) }

func dependencyTargetID(dep model.Dependency) string {
	if dep.TargetApplicationID > 0 {
		return appNodeID(dep.TargetApplicationID)
	}
	if dep.TargetComponentID > 0 {
		return componentNodeID(dep.TargetComponentID)
	}
	if dep.TargetResourceID > 0 {
		return resourceNodeID(dep.TargetResourceID)
	}
	return externalNodeID(dep.ID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "未命名"
}

func applicationTopologyMetadata(app model.Application) map[string]interface{} {
	metadata := map[string]interface{}{
		"code": app.Code, "owner": app.OwnerName, "team": app.Team, "criticality": app.Criticality,
		"lifecycle": app.Lifecycle, "language": app.Language, "repositoryUrl": app.RepositoryURL,
	}
	var tags []string
	if json.Unmarshal([]byte(app.Tags), &tags) == nil && len(tags) > 0 {
		metadata["tags"] = tags
	}
	return metadata
}

// MetadataJSON keeps topology metadata serializable for future graph clients.
func MetadataJSON(value map[string]interface{}) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
