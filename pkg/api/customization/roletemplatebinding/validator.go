package roletemplatebinding

import (
	"fmt"
	"net/http"

	"github.com/rancher/norman/httperror"
	"github.com/rancher/norman/types"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	client "github.com/rancher/types/client/management/v3"
	"github.com/rancher/types/config"
)

func NewPRTBValidator(management *config.ScaledContext) types.Validator {
	return newValidator(management, client.ProjectRoleTemplateBindingFieldRoleTemplateID, "project")
}

func NewCRTBValidator(management *config.ScaledContext) types.Validator {
	return newValidator(management, client.ClusterRoleTemplateBindingFieldRoleTemplateID, "cluster")
}

func newValidator(management *config.ScaledContext, field string, context string) types.Validator {
	validator := &validator{
		roleTemplateLister: management.Management.RoleTemplates("").Controller().Lister(),
		field:              field,
		context:            context,
	}

	return validator.validator
}

type validator struct {
	roleTemplateLister v3.RoleTemplateLister
	field              string
	context            string
}

func (v *validator) validator(request *types.APIContext, schema *types.Schema, data map[string]interface{}) error {
	roleTemplateName := data[v.field]
	if roleTemplateName == nil && request.Method == http.MethodPut {
		return nil
	}

	roleTemplate, err := v.validateRoleTemplateBinding(roleTemplateName)
	if err != nil {
		return err
	}

	if roleTemplate.Context != v.context {
		return httperror.NewAPIError(httperror.InvalidBodyContent, fmt.Sprintf("Cannot reference context [%s] from [%s] context",
			roleTemplate.Context, v.context))
	}

	if request.Method == http.MethodPut {
		return nil
	}

	hasUserTarget := data["userId"] != nil || data["userPrincipalId"] != nil
	hasGroupTarget := data["groupId"] != nil || data["groupPrincipalId"] != nil

	if (hasUserTarget && hasGroupTarget) || (!hasUserTarget && !hasGroupTarget) {
		return httperror.NewAPIError(httperror.InvalidBodyContent, "must target a user [userId]/[userPrincipalId] "+
			"OR a group [groupId]/[groupPrincipalId]")
	}

	return nil
}

func (v *validator) validateRoleTemplateBinding(obj interface{}) (*v3.RoleTemplate, error) {
	roleTemplateID, ok := obj.(string)
	if !ok {
		return nil, httperror.NewAPIError(httperror.MissingRequired, "Request does not have a valid roleTemplateId")
	}

	roleTemplate, err := v.roleTemplateLister.Get("", roleTemplateID)
	if err != nil {
		return nil, httperror.NewAPIError(httperror.ServerError, fmt.Sprintf("Error getting role template: %v", err))
	}

	if roleTemplate.Locked {
		return nil, httperror.NewAPIError(httperror.InvalidState, "Role is locked and cannot be assigned")
	}

	return roleTemplate, nil
}
