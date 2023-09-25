package elastic

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mycujoo/go-stdlib/pkg/kqlfilter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertNodeToQuery(t *testing.T) {
	testCases := []struct {
		name              string
		input             string
		expectedError     error
		expectedQueryJSON string
	}{
		{
			name:              "simple equality",
			input:             "type_id:team",
			expectedError:     nil,
			expectedQueryJSON: `{"term":{"type_id":{"value":"team"}}}`,
		},
		{
			name:              "multiple values for same field",
			input:             "type_id:(team OR player)",
			expectedError:     nil,
			expectedQueryJSON: `{"terms":{"type_id":["team","player"]}}`,
		},
		{
			name:          "multiple fields",
			input:         "type_id:team fields.active:true",
			expectedError: nil,
			expectedQueryJSON: `{
  "bool": {
    "must": [
      {
        "term": {
          "type_id": {
            "value": "team"
          }
        }
      },
	  {
        "term": {
          "fields.active": {
            "value": "true"
          }
        }
      }
    ]
  }
}`,
		},
		{
			name:          "and/or",
			input:         "type_id:team fields.active:true or fields.established_year < 2000",
			expectedError: nil,
			expectedQueryJSON: `{
  "bool": {
    "must": [
      {
        "term": {
          "type_id": {
            "value": "team"
          }
        }
      },
      {
        "bool": {
          "should": [
            {
              "term": {
                "fields.active": {
                  "value": "true"
                }
              }
            },
            {
              "range": {
                "fields.established_year": {
                  "lt": 2000
                }
              }
            }
          ]
        }
      }
    ]
  }
}`,
		},
		{
			name:          "not",
			input:         "not type_id:team",
			expectedError: nil,
			expectedQueryJSON: `{
  "bool": {
    "must_not": [
      {
        "term": {
          "type_id": {
            "value": "team"
          }
        }
      }
    ]
  }
}`,
		},
		{
			name:          "nested",
			input:         "type_id:player fields:{position:(goalkeeper OR defender)}",
			expectedError: nil,
			expectedQueryJSON: `{
  "bool": {
    "must": [
      {
        "term": {
          "type_id": {
            "value": "player"
          }
        }
      },
      {
        "terms": {
          "fields.position": [
            "goalkeeper",
            "defender"
          ]
        }
      }
    ]
  }
}`,
		},
		{
			name:          "range date",
			input:         `type_id:player fields.birthday >= "2000-01-01T00:00:00.000Z"`,
			expectedError: nil,
			expectedQueryJSON: `{
	  "bool": {
		"must": [	
		  {
			"term": {
			  "type_id": {	
				"value": "player"	
			  }
			}
},
		  {
			"range": {
			  "fields.birthday": {
				"gte": "2000-01-01T00:00:00.000Z"
			  }
			}
		  }
		]
	  }
}`,
		},
		{
			name:          "range invalid",
			input:         `type_id:player fields.birthday>=true`,
			expectedError: errors.New("fields.birthday: expected int or date literal"),
		},
		{
			name:          "nesting invalid",
			input:         `type_id:player fields.birthday:(value:"2000-01-01T00:00:00.000Z")`,
			expectedError: errors.New("fields.birthday: expected literal node"),
		},
		{
			name:          "invalid field",
			input:         `type:player`,
			expectedError: errors.New("type: invalid field"),
		},
		{
			name:          "invalid multiple values",
			input:         `type_id:(player OR team OR (club OR organization))`,
			expectedError: errors.New("type_id: invalid syntax"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			n, err := kqlfilter.ParseAST(test.input)
			require.NoError(t, err)

			g := NewQueryGenerator(WithFieldValidator(
				func(field string) error {
					if field == "type_id" {
						return nil
					}
					if strings.HasPrefix(field, "fields.") && strings.Count(field, ".") == 1 {
						return nil
					}
					return errors.New("invalid field")
				}))

			q, err := g.ConvertAST(n)
			if err != nil {
				require.EqualError(t, err, test.expectedError.Error())
				return
			}
			require.NoError(t, err)

			data, err := json.Marshal(q)
			require.NoError(t, err)

			assert.JSONEq(t, test.expectedQueryJSON, string(data))
		})
	}
}
