package role_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/abergmeier/terraform-provider-exasol/internal"
	"github.com/abergmeier/terraform-provider-exasol/internal/exaprovider"
	"github.com/abergmeier/terraform-provider-exasol/internal/test"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	roleSuffix = acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
)

func TestAccExasolRole_rename(t *testing.T) {

	dbName := fmt.Sprintf("%s_%s", t.Name(), roleSuffix)

	renamedDbName := fmt.Sprintf("%s_RENAMED", dbName)

	ps := test.NewDefaultAccProviders()
	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          nil,
		ProviderFactories: ps.Factories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`%s
				resource "exasol_role" "test_role" {
					name = "%s"
				}
				`, test.HCLProviderFromConf(exaConf), dbName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("exasol_role.test_role", "name", dbName),
					testExist(ps.Exasol, "exasol_role.test_role"),
				),
			},
			{
				Config: fmt.Sprintf(`%s
				resource "exasol_role" "test_role" {
					name = "%s"
				}
				`, test.HCLProviderFromConf(exaConf), renamedDbName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("exasol_role.test_role", "name", renamedDbName),
					testExist(ps.Exasol, "exasol_role.test_role"),
					testExistsNotByName(ps.Exasol, dbName),
				),
			},
		},
	})
}

func TestAccExasolRole_import(t *testing.T) {

	dbName := fmt.Sprintf("%s_%s", t.Name(), roleSuffix)

	locked := exaClient.Lock()
	defer locked.Unlock()
	tryDeleteRole := func() {
		stmt := fmt.Sprintf(`DROP ROLE %s`, dbName)
		_, err := locked.Conn.Execute(stmt)
		if err != nil {
			return
		}
		locked.Conn.Commit()
	}
	defer tryDeleteRole()

	ps := test.NewDefaultAccProviders()

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: ps.Factories,
		Steps: []resource.TestStep{
			{
				PreConfig: tryDeleteRole,
				Config: fmt.Sprintf(`%s
				resource "exasol_role" "test" {
					name = "%s"
				}
				`, test.HCLProviderFromConf(exaConf), strings.ToUpper(dbName)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("exasol_role.test", "name", strings.ToUpper(dbName)),
					testExist(ps.Exasol, "exasol_role.test"),
				),
			},
			{
				ResourceName:      "exasol_role.test",
				ImportState:       true,
				ImportStateId:     strings.ToUpper(dbName),
				ImportStateVerify: true,
			},
		},
	})
}

// exists checks whether the Role exists
func exists(c internal.Conn, name string) (bool, error) {
	res, err := c.FetchSlice("SELECT ROLE_NAME FROM EXA_ALL_ROLES WHERE UPPER(ROLE_NAME) = UPPER(?)", []interface{}{
		name,
	}, "SYS")
	if err != nil {
		return false, err
	}

	return len(res) != 0, nil
}

func testExistsNotByName(p *schema.Provider, actualName string) resource.TestCheckFunc {

	return func(state *terraform.State) error {

		c := p.Meta().(*exaprovider.Client)
		locked := c.Lock()
		defer locked.Unlock()

		exists, err := exists(locked.Conn, actualName)
		if err != nil {
			return err
		}

		if exists {
			return fmt.Errorf("Role %s does exist", actualName)
		}

		return nil
	}
}

func testExist(p *schema.Provider, id string) resource.TestCheckFunc {

	return func(state *terraform.State) error {

		rs, err := rootRole(state, id)
		if err != nil {
			return err
		}

		actualName, ok := rs.Primary.Attributes["name"]
		if !ok {
			return fmt.Errorf("Attribute name not found")
		}

		c := p.Meta().(*exaprovider.Client)
		locked := c.Lock()
		defer locked.Unlock()

		exists, err := exists(locked.Conn, actualName)
		if err != nil {
			return err
		}

		if !exists {
			return fmt.Errorf("Role %s does not exist", actualName)
		}

		return nil
	}
}

func testName(id, expectedName string) resource.TestCheckFunc {

	return func(state *terraform.State) error {

		rs, err := rootRole(state, id)
		if err != nil {
			return err
		}

		actualName := rs.Primary.Attributes["name"]
		if actualName != expectedName {
			return fmt.Errorf("Expected name %s: %s", expectedName, actualName)
		}

		return nil
	}
}

func rootRole(state *terraform.State, id string) (*terraform.ResourceState, error) {

	rs, ok := state.RootModule().Resources[id]
	if !ok {
		return nil, fmt.Errorf("Role not found: %s", id)
	}

	return rs, nil
}
