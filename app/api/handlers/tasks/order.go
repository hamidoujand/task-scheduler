package tasks

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/hamidoujand/task-scheduler/business/domain/task"
)

func parseOrder(r *http.Request) (task.OrderBy, error) {
	//default order by
	deafultOrder := task.OrderBy{
		Field:     task.FieldCreatedAt,
		Direction: task.DirectionASC,
	}

	orderString := r.URL.Query().Get("orderby")
	if orderString == "" {
		return deafultOrder, nil
	}

	parts := strings.Split(orderString, ",")

	var order task.OrderBy

	switch len(parts) {
	case 1:
		fieldString := parts[0]
		field, err := task.ParseField(fieldString)
		if err != nil {
			return task.OrderBy{}, fmt.Errorf("unknown field: %q", fieldString)
		}
		order.Field = field
		order.Direction = task.DirectionASC
	case 2:
		fieldString := parts[0]

		field, err := task.ParseField(fieldString)
		if err != nil {
			return task.OrderBy{}, fmt.Errorf("unknown field: %q", fieldString)
		}

		dirString := parts[1]

		dir, err := task.ParseDirection(dirString)
		if err != nil {
			return task.OrderBy{}, fmt.Errorf("unknown direction: %q", dirString)
		}

		order.Field = field
		order.Direction = dir
	}

	return order, nil
}
