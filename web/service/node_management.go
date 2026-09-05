package service

import (
	"errors"
	"net/url"
	"strings"

	"x-ui/database"
	"x-ui/database/model"

	"golang.org/x/crypto/bcrypt"
)

type NodeInput struct {
	Name     string `json:"name"`
	BaseURL  string `json:"baseUrl"`
	APIToken string `json:"apiToken"`
	Enabled  bool   `json:"enabled"`
}

func ValidateNodeInput(input NodeInput) error {
	if err := ValidateNodeURLAndName(input); err != nil {
		return err
	}
	if len(input.APIToken) < 16 {
		return errors.New("API token must be at least 16 characters")
	}
	return nil
}

func ValidateNodeUpdateInput(input NodeInput) error {
	if err := ValidateNodeURLAndName(input); err != nil {
		return err
	}
	if input.APIToken != "" && len(input.APIToken) < 16 {
		return errors.New("API token must be at least 16 characters")
	}
	return nil
}

func ValidateNodeURLAndName(input NodeInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("node name is required")
	}
	u, err := url.Parse(strings.TrimSpace(input.BaseURL))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("node base URL must be an HTTPS URL")
	}
	return nil
}

type NodePublic struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	BaseURL          string `json:"baseUrl"`
	Enabled          bool   `json:"enabled"`
	LastHealthStatus string `json:"lastHealthStatus"`
}

func ToNodePublic(node model.Node) NodePublic {
	return NodePublic{ID: node.ID, Name: node.Name, BaseURL: node.BaseURL, Enabled: node.Enabled, LastHealthStatus: node.LastHealthStatus}
}

func hashNodeToken(token string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	return string(hash), err
}

type NodeService struct{}

func (s *NodeService) List() ([]NodePublic, error) {
	var nodes []model.Node
	if err := database.GetDB().Order("id desc").Find(&nodes).Error; err != nil {
		return nil, err
	}
	result := make([]NodePublic, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, ToNodePublic(node))
	}
	return result, nil
}

func (s *NodeService) Create(input NodeInput) (*NodePublic, error) {
	if err := ValidateNodeInput(input); err != nil {
		return nil, err
	}
	hash, err := hashNodeToken(input.APIToken)
	if err != nil {
		return nil, err
	}
	node := &model.Node{Name: strings.TrimSpace(input.Name), BaseURL: strings.TrimRight(strings.TrimSpace(input.BaseURL), "/"), APITokenHash: hash, Enabled: input.Enabled}
	if err := database.GetDB().Create(node).Error; err != nil {
		return nil, err
	}
	result := ToNodePublic(*node)
	return &result, nil
}

func (s *NodeService) Update(id int, input NodeInput) error {
	if id <= 0 {
		return errors.New("node ID is required")
	}
	if err := ValidateNodeUpdateInput(input); err != nil {
		return err
	}
	updates := map[string]interface{}{
		"name":     strings.TrimSpace(input.Name),
		"base_url": strings.TrimRight(strings.TrimSpace(input.BaseURL), "/"),
		"enabled":  input.Enabled,
	}
	if input.APIToken != "" {
		hash, err := hashNodeToken(input.APIToken)
		if err != nil {
			return err
		}
		updates["api_token_hash"] = hash
	}
	result := database.GetDB().Model(&model.Node{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	return requireRowsAffected("node", result.RowsAffected)
}

func (s *NodeService) Delete(id int) error {
	if id <= 0 {
		return errors.New("node ID is required")
	}
	result := database.GetDB().Delete(&model.Node{}, id)
	if result.Error != nil {
		return result.Error
	}
	return requireRowsAffected("node", result.RowsAffected)
}
