package project

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type Project struct {
	Path     string   `json:"path"`
	Sessions []string `json:"sessions"`
}

type ProjectManager struct {
	configFile string
	projects   map[string]*Project
}

func NewProjectManager(projectRoot string) *ProjectManager {
	configFile := filepath.Join(projectRoot, ".projects/", "projects.json")

	pm := &ProjectManager{
		configFile: configFile,
		projects:   make(map[string]*Project),
	}
	pm.load()
	return pm
}

func (pm *ProjectManager) load() {
	data, err := os.ReadFile(pm.configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Failed to read project config: %v", err)
		}
		return
	}

	if len(data) == 0 {
		return
	}

	var container struct {
		Projects map[string]*Project `json:"projects"`
	}
	if err := json.Unmarshal(data, &container); err != nil {
		log.Printf("Warning: Failed to parse projects.json: %v", err)
		return
	}
	if container.Projects != nil {
		pm.projects = container.Projects
	}
}

func (pm *ProjectManager) save() {
	container := struct {
		Projects map[string]*Project `json:"projects"`
	}{
		Projects: pm.projects,
	}
	data, _ := json.MarshalIndent(container, "", "  ")
	os.MkdirAll(filepath.Dir(pm.configFile), 0755)
	os.WriteFile(pm.configFile, data, 0644)
}

func (pm *ProjectManager) GetOrCreate(path string) *Project {
	if project, exists := pm.projects[path]; exists {
		return project
	}

	project := &Project{
		Path:     path,
		Sessions: []string{},
	}
	pm.projects[path] = project
	pm.save()
	return project
}

func (pm *ProjectManager) AddSession(projectPath, sessionID string) {
	project := pm.GetOrCreate(projectPath)
	for _, id := range project.Sessions {
		if id == sessionID {
			return
		}
	}
	project.Sessions = append(project.Sessions, sessionID)
	pm.save()
}

func (pm *ProjectManager) GetProject(path string) *Project {
	return pm.projects[path]
}

func (pm *ProjectManager) ListProjects() []*Project {
	var list []*Project
	for _, p := range pm.projects {
		list = append(list, p)
	}
	return list
}
