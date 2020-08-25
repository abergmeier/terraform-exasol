package table_test

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/abergmeier/terraform-exasol/internal/datasources/test"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"github.com/grantstreetgroup/go-exasol-client"
)

var (
	tableSuffix = acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	testDefs    = []tableTest{
		{
			resourceName: "t1",
			tableName:    "t1_" + tableSuffix,
			stmt: fmt.Sprintf(`CREATE TABLE t1_%s (a VARCHAR(20),
			b DECIMAL(24,4) NOT NULL,
			c DECIMAL DEFAULT 122,
			d DOUBLE,
			e TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			f BOOL)`, tableSuffix),
			expectedColumns: []expectedColumns{
				{
					name: "A",
					t:    "VARCHAR(20) UTF8",
				},
				{
					name: "B",
					t:    "DECIMAL(24,4)",
				}, // NOT NULL,
				{
					name: "C",
					t:    "DECIMAL(18,0)",
				}, // DECIMAL DEFAULT 122,
				{
					name: "D",
					t:    "DOUBLE",
				},
				{
					name: "E",
					t:    "TIMESTAMP",
				}, //  DEFAULT CURRENT_TIMESTAMP,
				{
					name: "F",
					t:    "BOOLEAN",
				},
			},
		},
		{
			resourceName: "t2",
			tableName:    "t2_" + tableSuffix,
			stmt:         fmt.Sprintf(`CREATE TABLE t2_%s AS SELECT a,b,c+1 AS c FROM t1_%s`, tableSuffix, tableSuffix),
			expectedColumns: []expectedColumns{
				{
					name: "A",
					t:    "VARCHAR(20) UTF8",
				},
				{
					name: "B",
					t:    "DECIMAL(24,4)",
				}, //  NOT NULL,
				{
					name: "C",
					t:    "DECIMAL(19,0)",
				}, //  DEFAULT 122,
			},
		},
		{
			resourceName: "t3",
			tableName:    "t3_" + tableSuffix,
			stmt:         fmt.Sprintf(`CREATE TABLE t3_%s AS SELECT count(*) AS my_count FROM t1_%s WITH NO DATA`, tableSuffix, tableSuffix),
			expectedColumns: []expectedColumns{
				{
					name: "MY_COUNT",
					t:    "DECIMAL(18,0)",
				},
			},
		},
		{
			resourceName: "t4",
			tableName:    "t4_" + tableSuffix,
			stmt:         fmt.Sprintf(`CREATE TABLE t4_%s LIKE t1_%s`, tableSuffix, tableSuffix),
			expectedColumns: []expectedColumns{
				{
					name: "A",
					t:    "VARCHAR(20) UTF8",
				},
				{
					name: "B", //  NOT NULL,
					t:    "DECIMAL(24,4)",
				},
				{
					name: "C", //  DEFAULT 122,
					t:    "DECIMAL(18,0)",
				},
				{
					name: "D",
					t:    "DOUBLE",
				},
				{
					name: "E", //  DEFAULT CURRENT_TIMESTAMP,
					t:    "TIMESTAMP",
				},
				{
					name: "F",
					t:    "BOOLEAN",
				},
			},
		},
		{
			resourceName: "t5",
			tableName:    "t5_" + tableSuffix,
			stmt: fmt.Sprintf(`CREATE TABLE t5_%s (id int IDENTITY PRIMARY KEY,
				LIKE t1_%s INCLUDING DEFAULTS,
				g DOUBLE,
				DISTRIBUTE BY a,b)`, tableSuffix, tableSuffix),
			expectedColumns: []expectedColumns{
				{
					name: "ID",
					t:    "DECIMAL(18,0)",
				},
				{
					name: "A",
					t:    "VARCHAR(20) UTF8",
				},
				{
					name: "B",
					t:    "DECIMAL(24,4)",
				},
				{
					name: "C",
					t:    "DECIMAL(18,0)",
				},
				{
					name: "D",
					t:    "DOUBLE",
				},
				{
					name: "E",
					t:    "TIMESTAMP",
				},
				{
					name: "F",
					t:    "BOOLEAN",
				},
				{
					name: "G",
					t:    "DOUBLE",
				},
			},
		},
		{
			resourceName: "t6",
			tableName:    "t6_" + tableSuffix,
			stmt: fmt.Sprintf(`CREATE TABLE t6_%s (order_id INT,
					order_price DOUBLE,
					order_date DATE,
					country VARCHAR(40),
					PARTITION BY order_date)`, tableSuffix),
			expectedColumns: []expectedColumns{
				{
					name: "ORDER_ID",
					t:    "DECIMAL(18,0)",
				},
				{
					name: "ORDER_PRICE",
					t:    "DOUBLE",
				},
				{
					name: "ORDER_DATE",
					t:    "DATE",
				},
				{
					name: "COUNTRY",
					t:    "VARCHAR(40) UTF8",
				},
			},
		},
		{
			resourceName: "t7",
			tableName:    "t7_" + tableSuffix,
			stmt:         fmt.Sprintf(`SELECT * INTO TABLE t7_%s FROM t1_%s`, tableSuffix, tableSuffix),
			expectedColumns: []expectedColumns{
				{
					name: "A",
					t:    "VARCHAR(20) UTF8",
				},
				{
					name: "B",
					t:    "DECIMAL(24,4)", // NOT NULL,
				},
				{
					name: "C",
					t:    "DECIMAL(18,0)", // DEFAULT 122,
				},
				{
					name: "D",
					t:    "DOUBLE",
				},
				{
					name: "E",
					t:    "TIMESTAMP", // DEFAULT CURRENT_TIMESTAMP,
				},
				{
					name: "F",
					t:    "BOOLEAN",
				},
			},
		},
		{
			resourceName: "t8",
			tableName:    "t8_" + tableSuffix,
			stmt:         fmt.Sprintf(`CREATE TABLE t8_%s (ref_id int CONSTRAINT FK_T5 REFERENCES t5_%s (id) DISABLE, b VARCHAR(20))`, tableSuffix, tableSuffix),
			expectedColumns: []expectedColumns{
				{
					name: "REF_ID",
					t:    "DECIMAL(18,0)",
				},
				{
					name: "B",
					t:    "VARCHAR(20) UTF8",
				},
			},
		},
	}
)

type tableTest struct {
	resourceName    string
	tableName       string
	stmt            string
	config          string
	expectedColumns []expectedColumns
}

type expectedColumns struct {
	name string
	t    string
}

// TestAccExasolTable_basic all examples provided by Exasol.
func TestAccExasolTable_basic(t *testing.T) {
	locked := exaClient.Lock()
	defer locked.Unlock()

	for i, v := range testDefs {
		testDefs[i].config = fmt.Sprintf(`%s
data "exasol_physical_schema" "dummy" {
	name = "%s"
}
data "exasol_table" "%s" {
	name = "%s"
	schema = data.exasol_physical_schema.dummy.name
}
`, test.ProviderInHCL(locked), schemaName, v.resourceName, v.tableName)
	}

	basicSetup(t, locked.Conn)

	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: test.DefaultAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDefs[0].config,
				Check: resource.ComposeTestCheckFunc(
					testColumns("data.exasol_table.t1", testDefs[0].expectedColumns),
					testPrimaryKeys("data.exasol_table.t1", nil),
					testForeignKeys("data.exasol_table.t1", nil),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: test.DefaultAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDefs[1].config,
				Check: resource.ComposeTestCheckFunc(
					testColumns("data.exasol_table.t2", testDefs[1].expectedColumns),
					testPrimaryKeys("data.exasol_table.t2", nil),
					testForeignKeys("data.exasol_table.t2", nil),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: test.DefaultAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDefs[2].config,
				Check: resource.ComposeTestCheckFunc(
					testColumns("data.exasol_table.t3", testDefs[2].expectedColumns),
					testPrimaryKeys("data.exasol_table.t3", nil),
					testForeignKeys("data.exasol_table.t3", nil),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: test.DefaultAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDefs[3].config,
				Check: resource.ComposeTestCheckFunc(
					testColumns("data.exasol_table.t4", testDefs[3].expectedColumns),
					testPrimaryKeys("data.exasol_table.t4", nil),
					testForeignKeys("data.exasol_table.t4", nil),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: test.DefaultAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDefs[4].config,
				Check: resource.ComposeTestCheckFunc(
					testColumns("data.exasol_table.t5", testDefs[4].expectedColumns),
					testPrimaryKeys("data.exasol_table.t5", map[string]int{
						"id": 0,
					}),
					testForeignKeys("data.exasol_table.t5", nil),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: test.DefaultAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDefs[5].config,
				Check: resource.ComposeTestCheckFunc(
					testColumns("data.exasol_table.t6", testDefs[5].expectedColumns),
					testPrimaryKeys("data.exasol_table.t6", nil),
					testForeignKeys("data.exasol_table.t6", nil),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: test.DefaultAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDefs[6].config,
				Check: resource.ComposeTestCheckFunc(
					testColumns("data.exasol_table.t7", testDefs[6].expectedColumns),
					testPrimaryKeys("data.exasol_table.t7", nil),
					testForeignKeys("data.exasol_table.t7", nil),
				),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		PreCheck:  nil,
		Providers: test.DefaultAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testDefs[7].config,
				Check: resource.ComposeTestCheckFunc(
					testColumns("data.exasol_table.t8", testDefs[7].expectedColumns),
					testPrimaryKeys("data.exasol_table.t8", nil),
					testForeignKeys("data.exasol_table.t8", map[string]int{
						"ref_id": 0,
					}),
				),
			},
		},
	})
}

func basicSetup(t *testing.T, c *exasol.Conn) {

	for _, testDef := range testDefs {

		stmt := testDef.stmt

		tryDropTable(testDef.tableName, c)

		_, err := c.Execute(stmt, nil, schemaName)
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
	}
	c.Commit()
}

func tryDropTable(ref string, c *exasol.Conn) {
	stmt := fmt.Sprintf("DROP TABLE %s", ref)
	c.Execute(stmt, nil, schemaName)
}

func testColumns(id string, expected []expectedColumns) resource.TestCheckFunc {

	return func(state *terraform.State) error {

		ds, err := testDatasource(state, id)
		if err != nil {
			return err
		}

		countString, ok := ds.Primary.Attributes["columns.#"]
		if !ok {
			return fmt.Errorf("Column count not found: %s", id)
		}
		count, err := strconv.Atoi(countString)
		if err != nil {
			return err
		}

		if len(expected) != count {
			return fmt.Errorf("Expected %d elements: %d", len(expected), count)
		}

		for i := 0; i != count; i++ {
			e := expected[i]

			name := ds.Primary.Attributes[fmt.Sprintf("columns.%d.name", i)]
			if name != e.name {
				return fmt.Errorf("Name mismatch at %d. Expected %s: %s", i, e.name, name)
			}
			ct := ds.Primary.Attributes[fmt.Sprintf("columns.%d.type", i)]
			if ct != e.t {
				return fmt.Errorf("Type mismatch at %d. Expected %s: %s", i, e.t, ct)
			}
		}
		return nil
	}
}

func testDatasource(state *terraform.State, id string) (*terraform.ResourceState, error) {

	ds, ok := state.RootModule().Resources[id]
	if !ok {
		return nil, fmt.Errorf("Datasource not found: %s", id)
	}

	return ds, nil
}

func testForeignKeys(id string, expected map[string]int) resource.TestCheckFunc {

	if expected == nil {
		expected = map[string]int{}
	}

	return func(state *terraform.State) error {
		ds, err := testDatasource(state, id)
		if err != nil {
			return err
		}

		actual := map[string]int{}

		for k, v := range ds.Primary.Attributes {
			if strings.HasPrefix(k, "foreign_key_indices.") && !strings.HasSuffix(k, ".%") {
				actualIndex, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				nonPrefixedKey := k[len("foreign_key_indices."):]
				actual[nonPrefixedKey] = actualIndex
			}
		}

		if !reflect.DeepEqual(&actual, &expected) {
			return fmt.Errorf(`Foreign Key mismatch:
	Expected %#v
	Actual   %#v`, expected, actual)
		}

		return nil
	}
}

func testPrimaryKeys(id string, expected map[string]int) resource.TestCheckFunc {

	if expected == nil {
		expected = map[string]int{}
	}

	return func(state *terraform.State) error {

		ds, err := testDatasource(state, id)
		if err != nil {
			return err
		}

		actual := map[string]int{}

		for k, v := range ds.Primary.Attributes {
			if strings.HasPrefix(k, "primary_key_indices.") && !strings.HasSuffix(k, ".%") {
				actualIndex, err := strconv.Atoi(v)
				if err != nil {
					return err
				}
				nonPrefixedKey := k[len("primary_key_indices."):]
				actual[nonPrefixedKey] = actualIndex
			}
		}

		if !reflect.DeepEqual(&actual, &expected) {
			return fmt.Errorf(`Primary Key mismatch:
	Expected %#v
	Actual   %#v`, expected, actual)
		}

		return nil
	}
}