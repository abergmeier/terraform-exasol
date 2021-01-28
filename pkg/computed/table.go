package computed

import (
	"fmt"
	"strings"

	"github.com/grantstreetgroup/go-exasol-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type tableColumns struct {
	cols        []interface{}
	indices     map[string]interface{}
	distributes []string
}

type TableReader struct {
	Columns       []interface{}
	ColumnIndices map[string]interface{}
	Composite     string
	PrimaryKeys   map[string]interface{}
	ForeignKeys   map[string]interface{}
	distributes   []string
}

// ColumnIndicesSchema provides a fully computed Schema for Column Indices of a Table
func ColumnIndicesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Computed: true,
		Elem: &schema.Schema{
			Type: schema.TypeInt,
		},
	}
}

// ColumnsSchema provides a fully computed Schema for Columns of a Table
func ColumnsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": {
					Type:     schema.TypeString,
					Computed: true,
				},
				"type": {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	}
}

// ForeignKeysSchema provides a fully computed Schema for Foreign Keys of a Table
func ForeignKeysSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Computed: true,
		Elem: &schema.Schema{
			Type: schema.TypeInt,
		},
	}
}

// PrimaryKeysSchema provides a fully computed Schema for Primary Keys of a Table
func PrimaryKeysSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Computed: true,
		Elem: &schema.Schema{
			Type: schema.TypeInt,
		},
	}
}

// ReadTable reads necessary information of a Table
func ReadTable(c *exasol.Conn, schema, table string) (*TableReader, error) {
	tr := &TableReader{}
	var err error
	tcs, err := readColumns(c, schema, table)
	if err != nil {
		return nil, err
	}
	tr.Columns = tcs.cols
	tr.ColumnIndices = tcs.indices
	tr.PrimaryKeys, err = readPrimaryKeys(c, schema, table)
	if err != nil {
		return nil, err
	}
	tr.ForeignKeys, err = readForeignKeys(c, schema, table)
	if err != nil {
		return nil, err
	}

	stmt := `SELECT COLUMN_NAME, COLUMN_TYPE, COLUMN_IS_NULLABLE
FROM EXA_ALL_COLUMNS
WHERE UPPER(COLUMN_SCHEMA) = UPPER(?) AND UPPER(COLUMN_TABLE) = UPPER(?)
ORDER BY COLUMN_ORDINAL_POSITION`
	res, err := c.FetchSlice(stmt, []interface{}{
		schema,
		table,
	})
	if err != nil {
		return nil, err
	}

	b := &strings.Builder{}
	for _, column := range res {
		b.WriteString(column[0].(string))
		b.WriteString(" ")
		b.WriteString(column[1].(string))
		nullable := column[2].(bool)
		if nullable {
			b.WriteString(" NULL,\n")
		} else {
			b.WriteString(" NOT NULL,\n")
		}
	}
	for columnName := range tr.PrimaryKeys {
		fmt.Fprintf(b, "CONSTRAINT PRIMARY KEY (%s),\n", strings.ToUpper(columnName))
	}

	if len(tcs.distributes) > 0 {
		dl := strings.Join(tcs.distributes, ", ")
		fmt.Fprintf(b, "DISTRIBUTE BY %s,\n", strings.ToUpper(dl))
	}
	tr.Composite = b.String()
	return tr, nil
}

func readPrimaryKeys(c *exasol.Conn, schema, name string) (map[string]interface{}, error) {
	stmt := "SELECT COLUMN_NAME, ORDINAL_POSITION FROM EXA_ALL_CONSTRAINT_COLUMNS WHERE UPPER(CONSTRAINT_SCHEMA) = UPPER(?) AND UPPER(CONSTRAINT_TABLE) = UPPER(?) AND CONSTRAINT_TYPE = 'PRIMARY KEY'"
	cons, err := c.FetchSlice(stmt, []interface{}{
		schema,
		name,
	}, "SYS")
	if err != nil {
		return nil, err
	}

	pks := make(map[string]interface{}, len(cons))

	for _, values := range cons {
		name := values[0].(string)
		pks[strings.ToLower(name)] = int(values[1].(float64)+0.5) - 1
	}

	return pks, nil
}

func readForeignKeys(c *exasol.Conn, schema, name string) (map[string]interface{}, error) {
	stmt := "SELECT COLUMN_NAME, ORDINAL_POSITION FROM EXA_ALL_CONSTRAINT_COLUMNS WHERE UPPER(CONSTRAINT_SCHEMA) = UPPER(?) AND UPPER(CONSTRAINT_TABLE) = UPPER(?) AND CONSTRAINT_TYPE = 'FOREIGN KEY'"
	cons, err := c.FetchSlice(stmt, []interface{}{
		schema,
		name,
	}, "SYS")
	if err != nil {
		return nil, err
	}

	fks := make(map[string]interface{}, len(cons))

	for _, values := range cons {
		name := values[0].(string)
		fks[strings.ToLower(name)] = int(values[1].(float64)+0.5) - 1
	}

	return fks, nil
}

func readColumns(c *exasol.Conn, schema, table string) (tableColumns, error) {
	stmt := `SELECT COLUMN_ORDINAL_POSITION, COLUMN_NAME, COLUMN_TYPE, COLUMN_IS_DISTRIBUTION_KEY
		FROM EXA_ALL_COLUMNS
		WHERE UPPER(COLUMN_SCHEMA) = UPPER(?) AND UPPER(COLUMN_TABLE) = UPPER(?)
		ORDER BY COLUMN_ORDINAL_POSITION`

	res, err := c.FetchSlice(stmt, []interface{}{
		schema,
		table,
	}, "SYS")
	if err != nil {
		return tableColumns{}, err
	}

	tcs := tableColumns{
		cols:    make([]interface{}, len(res)),
		indices: make(map[string]interface{}, len(res)),
	}

	for i, values := range res {
		cn := values[1].(string)
		col := map[string]interface{}{
			"name": cn,
			"type": values[2].(string),
		}
		tcs.cols[i] = col
		isDistributionColumn := values[3].(bool)
		if isDistributionColumn {
			tcs.distributes = append(tcs.distributes, cn)
		}
		tcs.indices[strings.ToLower(cn)] = int(values[0].(float64)+0.5) - 1
	}

	return tcs, nil
}
