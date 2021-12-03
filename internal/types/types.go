package types

// Root struct for commands
// Field tags control where the values come from
// If opts:"-" yaml:"-" are set in object creation
//    opts:="-" come from config file
//    yaml:="-" come from command line flags
type Root struct {
}
