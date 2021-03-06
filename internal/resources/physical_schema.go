package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/abergmeier/terraform-provider-exasol/internal"
	"github.com/abergmeier/terraform-provider-exasol/internal/exaprovider"
	"github.com/abergmeier/terraform-provider-exasol/pkg/argument"
	"github.com/abergmeier/terraform-provider-exasol/pkg/db"
	"github.com/grantstreetgroup/go-exasol-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// PhysicalSchema returns the schema.Resource for managing a non-virtual Schema
func PhysicalSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of Schema",
			},
		},
		Create:      createPhysicalSchema,
		ReadContext: readPhysicalSchema,
		Update:      updatePhysicalSchema,
		Delete:      deletePhysicalSchema,
		Importer: &schema.ResourceImporter{
			State: importPhysicalSchema,
		},
	}
}

func createPhysicalSchema(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*exaprovider.Client)
	locked := c.Lock()
	defer locked.Unlock()
	err := createPhysicalSchemaData(d, locked.Conn)
	if err != nil {
		return err
	}
	return locked.Conn.Commit()
}

func createPhysicalSchemaData(d internal.Data, c *exasol.Conn) error {
	name, err := argument.Name(d)
	if err != nil {
		return err
	}

	stmt := fmt.Sprintf("CREATE SCHEMA %s", name)
	_, err = c.Execute(stmt)

	if err != nil {
		return err
	}

	d.SetId(strings.ToUpper(name))
	return nil
}

func deletePhysicalSchema(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*exaprovider.Client)
	locked := c.Lock()
	defer locked.Unlock()
	err := deletePhysicalSchemaData(d, locked.Conn)
	if err != nil {
		return err
	}
	return locked.Conn.Commit()
}

func deletePhysicalSchemaData(d internal.Data, c *exasol.Conn) error {
	name := d.Get("name").(string)

	stmt := fmt.Sprintf("DROP SCHEMA %s", name)
	_, err := c.Execute(stmt)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func importPhysicalSchema(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	c := meta.(*exaprovider.Client)
	locked := c.Lock()
	defer locked.Unlock()
	err := importPhysicalSchemaData(d, locked.Conn)
	if err != nil {
		return nil, err
	}
	err = locked.Conn.Commit()
	if err != nil {
		return nil, err
	}
	return []*schema.ResourceData{d}, nil
}

func importPhysicalSchemaData(d internal.Data, c *exasol.Conn) error {

	slice, err := c.FetchSlice("SELECT SCHEMA_NAME FROM EXA_SCHEMAS WHERE UPPER(SCHEMA_NAME) = UPPER(?) AND SCHEMA_IS_VIRTUAL = false", []interface{}{
		d.Id(),
	}, "SYS")
	if err != nil {
		return err
	}

	if len(slice) == 0 {
		return fmt.Errorf("Schema %s not found", d.Id())
	}
	d.SetId(strings.ToUpper(d.Id()))
	return nil
}

func readPhysicalSchema(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*exaprovider.Client)
	locked := c.Lock()
	defer locked.Unlock()
	return readPhysicalSchemaData(d, locked.Conn)
}

func readPhysicalSchemaData(d internal.Data, c *exasol.Conn) diag.Diagnostics {
	name, err := argument.Name(d)
	if err != nil {
		return diag.FromErr(err)
	}

	res, err := c.FetchSlice("SELECT SCHEMA_NAME FROM EXA_SCHEMAS WHERE UPPER(SCHEMA_NAME) = UPPER(?) AND SCHEMA_IS_VIRTUAL = FALSE ", []interface{}{
		name,
	}, "SYS")
	if err != nil {
		return diag.FromErr(err)
	}

	if len(res) == 0 {
		return diag.Errorf("Schema %s not found", name)
	}

	d.SetId(strings.ToUpper(name))
	return nil
}

func updatePhysicalSchema(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*exaprovider.Client)
	locked := c.Lock()
	defer locked.Unlock()
	err := updatePhysicalSchemaData(d, locked.Conn)
	if err != nil {
		return err
	}
	return locked.Conn.Commit()
}

func updatePhysicalSchemaData(d internal.Data, c *exasol.Conn) error {

	if d.HasChange("name") {
		old, new := d.GetChange("name")
		err := db.Rename(c, "SCHEMA", old.(string), new.(string), "")
		if err != nil {
			return err
		}

		d.Set("name", new)
	}

	_ = d.Get("name").(string)
	return nil
}
