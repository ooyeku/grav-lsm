package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ooyeku/grav-lsm/internal/model"
	"github.com/spf13/cobra"
	"os"
)

var modelManager *model.ModelManager

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage data models",
}

var createModelCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new model",
	Args:  cobra.ExactArgs(1),
	Run:   runCreateModel,
}

var updateModelCmd = &cobra.Command{
	Use:   "update [name]",
	Short: "Update an existing model",
	Args:  cobra.ExactArgs(1),
	Run:   runUpdateModel,
}

var listModelsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all models",
	Run:   runListModels,
}

var generateModelCmd = &cobra.Command{
	Use:   "generate [name]",
	Short: "Generate Go code for an existing model",
	Args:  cobra.ExactArgs(1),
	Run:   runGenerateModel,
}

func init() {
	modelManager = model.NewModelManager()

	createModelCmd.Flags().StringSlice("fields", []string{}, "Comma-separated list of fields in the format name:type")
	updateModelCmd.Flags().StringSlice("add-fields", []string{}, "Comma-separated list of fields to add in the format name:type")
	updateModelCmd.Flags().StringSlice("remove-fields", []string{}, "Comma-separated list of field names to remove")

	generateModelCmd.Flags().String("app", "", "Name of the Grav app to generate the model in")

	modelCmd.AddCommand(createModelCmd)
	modelCmd.AddCommand(updateModelCmd)
	RootCmd.AddCommand(modelCmd)
	modelCmd.AddCommand(listModelsCmd)
	modelCmd.AddCommand(generateModelCmd)
}

func runCreateModel(cmd *cobra.Command, args []string) {
	modelName := args[0]
	fields, _ := cmd.Flags().GetStringSlice("fields")

	modelFields, err := parseFields(fields)
	if err != nil {
		log.WithError(err).Error("Failed to parse fields")
		return
	}

	err = modelManager.CreateModel(modelName, modelFields)
	if err != nil {
		log.WithError(err).Errorf("Failed to create model %s", modelName)
		return
	}

	log.Infof("Model %s created successfully", modelName)
}

func runUpdateModel(cmd *cobra.Command, args []string) {
	modelName := args[0]
	addFields, _ := cmd.Flags().GetStringSlice("add-fields")
	removeFields, _ := cmd.Flags().GetStringSlice("remove-fields")

	modelDef, err := modelManager.GetModel(modelName)
	if err != nil {
		log.WithError(err).Errorf("Failed to get model %s", modelName)
		return
	}

	if len(addFields) > 0 {
		newFields, err := parseFields(addFields)
		if err != nil {
			log.WithError(err).Error("Failed to parse new fields")
			return
		}
		modelDef.Fields = append(modelDef.Fields, newFields...)
	}

	if len(removeFields) > 0 {
		modelDef.Fields = removeFieldsFromModel(modelDef.Fields, removeFields)
	}

	err = modelManager.UpdateModel(modelName, modelDef.Fields)
	if err != nil {
		log.WithError(err).Errorf("Failed to update model %s", modelName)
		return
	}

	err = model.GenerateModelFile(modelDef)
	if err != nil {
		log.WithError(err).Errorf("Failed to generate updated model file for %s", modelName)
		return
	}

	log.Infof("Model %s updated successfully", modelName)
}

func runListModels(cmd *cobra.Command, args []string) {
	models := modelManager.ListModels()
	if len(models) == 0 {
		log.Info("No models found.")
		return
	}

	log.Info("Available models:")
	for _, model := range models {
		log.Infof("- %s", model)
	}
}

func runGenerateModel(cmd *cobra.Command, args []string) {
	modelName := args[0]
	appName, _ := cmd.Flags().GetString("app")

	modelDef, err := modelManager.GetModel(modelName)
	if err != nil {
		log.WithError(err).Errorf("Failed to get model %s", modelName)
		return
	}

	var outputDir string
	if appName != "" {
		outputDir = filepath.Join(appName+"_grav", "internal", "models")
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.WithError(err).Errorf("Failed to create directory for app %s", appName)
			return
		}

		// Set the output directory for the model generator
		modelDef.SetOutputDir(outputDir)
	}

	err = model.GenerateModelFile(modelDef)
	if err != nil {
		log.WithError(err).Errorf("Failed to generate model file for %s", modelName)
		return
	}

	if appName != "" {
		log.Infof("Model file for %s generated successfully in %s", modelName, outputDir)
	} else {
		log.Infof("Model file for %s generated successfully", modelName)
	}
}

func parseFields(fields []string) ([]model.Field, error) {
	var modelFields []model.Field
	for _, field := range fields {
		parts := strings.Split(field, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field format: %s", field)
		}
		name := parts[0]
		fieldType := parts[1]
		tag := fmt.Sprintf(`json:"%s"`, strings.ToLower(name))
		isNull := false
		isPrimary := name == "ID" || name == "Id" || name == "id"
		modelFields = append(modelFields, model.NewField(name, fieldType, tag, isNull, isPrimary))
	}
	return modelFields, nil
}

func removeFieldsFromModel(fields []model.Field, fieldsToRemove []string) []model.Field {
	var updatedFields []model.Field
	for _, field := range fields {
		if !contains(fieldsToRemove, field.Name) {
			updatedFields = append(updatedFields, field)
		}
	}
	return updatedFields
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
