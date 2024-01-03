package v1alpha1

type NamespacedName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

const (
	Separator = '/'
)

// String returns the general purpose string representation
func (n NamespacedName) String() string {
	return n.Namespace + string(Separator) + n.Name
}
