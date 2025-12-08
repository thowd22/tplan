package main

import (
	"log"

	"github.com/yourusername/tplan/internal/models"
	"github.com/yourusername/tplan/internal/tui"
)

func main() {
	// Example: Create a sample Terraform plan
	plan := models.Plan{
		TerraformVersion: "1.5.0",
		FormatVersion:    "1.0",
		Resources: []models.Resource{
			{
				Address:      "aws_instance.web_server",
				Type:         "aws_instance",
				Name:         "web_server",
				Mode:         "managed",
				ProviderName: "aws",
				Change: models.Change{
					Actions: []string{"create"},
					Before:  nil,
					After: map[string]interface{}{
						"ami":           "ami-12345678",
						"instance_type": "t2.micro",
						"tags": map[string]interface{}{
							"Name": "WebServer",
							"Env":  "production",
						},
					},
				},
			},
			{
				Address:      "aws_s3_bucket.data",
				Type:         "aws_s3_bucket",
				Name:         "data",
				Mode:         "managed",
				ProviderName: "aws",
				Change: models.Change{
					Actions: []string{"update"},
					Before: map[string]interface{}{
						"bucket":        "my-data-bucket",
						"acl":           "private",
						"versioning":    false,
						"force_destroy": false,
					},
					After: map[string]interface{}{
						"bucket":        "my-data-bucket",
						"acl":           "private",
						"versioning":    true,
						"force_destroy": false,
					},
				},
			},
			{
				Address:      "aws_security_group.old_sg",
				Type:         "aws_security_group",
				Name:         "old_sg",
				Mode:         "managed",
				ProviderName: "aws",
				Change: models.Change{
					Actions: []string{"delete"},
					Before: map[string]interface{}{
						"name":        "old-security-group",
						"description": "Old security group to be removed",
						"vpc_id":      "vpc-12345",
					},
					After: nil,
				},
			},
			{
				Address:      "aws_db_instance.database",
				Type:         "aws_db_instance",
				Name:         "database",
				Mode:         "managed",
				ProviderName: "aws",
				Change: models.Change{
					Actions: []string{"delete", "create"}, // Replace
					Before: map[string]interface{}{
						"engine":         "postgres",
						"engine_version": "12.5",
						"instance_class": "db.t2.small",
					},
					After: map[string]interface{}{
						"engine":         "postgres",
						"engine_version": "14.2",
						"instance_class": "db.t3.medium",
					},
				},
			},
			{
				Address:      "aws_vpc.main",
				Type:         "aws_vpc",
				Name:         "main",
				Mode:         "managed",
				ProviderName: "aws",
				Change: models.Change{
					Actions: []string{"create"},
					Before:  nil,
					After: map[string]interface{}{
						"cidr_block":           "10.0.0.0/16",
						"enable_dns_hostnames": true,
						"enable_dns_support":   true,
						"tags": map[string]interface{}{
							"Name": "main-vpc",
						},
					},
				},
			},
		},
	}

	// Run the TUI
	if err := tui.Run(plan); err != nil {
		log.Fatal(err)
	}
}
