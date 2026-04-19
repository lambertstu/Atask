package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Project struct {
	Path         string    `json:"path"`
	Sessions     []string  `json:"sessions"`
	LastModified time.Time `json:"last_modified"`
}

type ProjectManager struct {
	configFile string
	mu         sync.RWMutex
}

func NewProjectManager(projectRoot string) *ProjectManager {
	configFile := filepath.Join(projectRoot, ".projects/", "projects.json")
	os.MkdirAll(filepath.Dir(configFile), 0755)

	return &ProjectManager{
		configFile: configFile,
	}
}

func (pm *ProjectManager) readAllFromFile() map[string]*Project {
	data, err := os.ReadFile(pm.configFile)
	if err != nil {
		return make(map[string]*Project)
	}

	if len(data) == 0 {
		return make(map[string]*Project)
	}

	var container struct {
		Projects map[string]*Project `json:"projects"`
	}
	if err := json.Unmarshal(data, &container); err != nil {
		return make(map[string]*Project)
	}
	if container.Projects == nil {
		return make(map[string]*Project)
	}
	for _, p := range container.Projects {
		if p.LastModified.IsZero() {
			p.LastModified = time.Now()
		}
	}
	return container.Projects
}

func (pm *ProjectManager) saveAll(projects map[string]*Project) {
	container := struct {
		Projects map[string]*Project `json:"projects"`
	}{
		Projects: projects,
	}
	data, _ := json.MarshalIndent(container, "", "  ")
	os.WriteFile(pm.configFile, data, 0644)
}

func (pm *ProjectManager) GetOrCreate(path string) *Project {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := filepath.Base(path)
	projects := pm.readAllFromFile()
	if project, exists := projects[key]; exists {
		return project
	}

	project := &Project{
		Path:         path,
		Sessions:     []string{},
		LastModified: time.Now(),
	}
	projects[key] = project
	pm.saveAll(projects)
	return project
}

func (pm *ProjectManager) OpenProject(path string) *Project {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := filepath.Base(path)
	projects := pm.readAllFromFile()
	if project, exists := projects[key]; exists {
		project.LastModified = time.Now()
		pm.saveAll(projects)
		return project
	}

	project := &Project{
		Path:         path,
		Sessions:     []string{},
		LastModified: time.Now(),
	}
	projects[key] = project
	pm.saveAll(projects)
	return project
}

func (pm *ProjectManager) AddSession(projectPath, sessionID string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := filepath.Base(projectPath)
	projects := pm.readAllFromFile()
	project, exists := projects[key]
	if !exists {
		project = &Project{
			Path:         projectPath,
			Sessions:     []string{},
			LastModified: time.Now(),
		}
		projects[key] = project
	}

	for _, id := range project.Sessions {
		if id == sessionID {
			return
		}
	}
	project.Sessions = append(project.Sessions, sessionID)
	project.LastModified = time.Now()
	pm.saveAll(projects)
}

func (pm *ProjectManager) GetProject(path string) *Project {
	projects := pm.readAllFromFile()
	key := filepath.Base(path)
	return projects[key]
}

func (pm *ProjectManager) ListProjects() []*Project {
	projects := pm.readAllFromFile()

	var list []*Project
	for _, p := range projects {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].LastModified.After(list[j].LastModified)
	})
	return list
}

func (pm *ProjectManager) RemoveSession(projectPath, sessionID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	key := filepath.Base(projectPath)
	projects := pm.readAllFromFile()
	project, exists := projects[key]
	if !exists {
		return fmt.Errorf("project not found: %s", key)
	}

	found := false
	newSessions := make([]string, 0, len(project.Sessions))
	for _, id := range project.Sessions {
		if id == sessionID {
			found = true
			continue
		}
		newSessions = append(newSessions, id)
	}

	if !found {
		return fmt.Errorf("session not found in project: %s", sessionID)
	}

	project.Sessions = newSessions
	project.LastModified = time.Now()
	pm.saveAll(projects)
	return nil
}
