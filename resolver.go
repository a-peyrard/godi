package godi

import (
	"errors"
	"fmt"
	"github.com/a-peyrard/godi/fn"
	"github.com/a-peyrard/godi/option"
	"reflect"
	"strings"
	"time"
)

const perfOutput = true

type (
	Query interface {
	}

	Name struct {
		name string
		typ  reflect.Type
	}

	Request struct {
		unitaryTyp reflect.Type
		query      query
		validator  validator
		collector  collector
		tracker    *Tracker
	}

	Resolver struct {
		providers *SortedCOWSlice[Provider]
		store     *Store

		lock *LockManager
	}

	// Closeable is an interface that can be used to close resources.
	Closeable interface {
		Close() error
	}

	Registrable = any

	RegistrableOptions struct {
		named        string
		priority     int
		dependencies []dependency
		conditions   []condition

		description string
	}
)

func Named(name string) option.Option[RegistrableOptions] {
	return func(opts *RegistrableOptions) {
		opts.named = name
	}
}

func Priority(priority int) option.Option[RegistrableOptions] {
	return func(opts *RegistrableOptions) {
		opts.priority = priority
	}
}

func Dependencies(dependencies ...dependency) option.Option[RegistrableOptions] {
	return func(opts *RegistrableOptions) {
		opts.dependencies = dependencies
	}
}

func Description(description string) option.Option[RegistrableOptions] {
	return func(opts *RegistrableOptions) {
		opts.description = description
	}
}

func (n Name) String() string {
	return fmt.Sprintf("(%s, %s)", n.name, n.typ.String())
}

func (r Request) String() string {
	return fmt.Sprintf("{q=%s v=%s c=%s}", r.query, r.validator, r.collector)
}

func New() *Resolver {

	r := &Resolver{
		providers: NewSortedCOWSlice[Provider](fn.ReverseComparator(compareByPriority)),
		store:     NewStore(),

		lock: NewLockManager(),
	}

	// Register itself as a static provider.
	//
	// If providers want to resolve the resolver to be able to dynamically resolve dependencies
	r.MustRegister(ToStaticProvider(r), Named("godi.resolver"))

	return r
}

func (r *Resolver) Register(reg Registrable, opts ...option.Option[RegistrableOptions]) error {
	var (
		t        = reflect.TypeOf(reg)
		provider Provider
		err      error
	)
	if t.Kind() == reflect.Func {
		provider, err = NewFactoryMethodProvider(reg, opts...)
		if err != nil {
			return fmt.Errorf("failed to create factory method provider for %T:\n\t%w", reg, err)
		}
	} else if t.Implements(ProviderType) {
		provider = reg.(Provider)
	} else {
		return errors.New("provider must be either a function or a Provider implementation")
	}

	options := option.Build(
		&RegistrableOptions{},
		opts...,
	)

	// validate the conditions if any, they might prevent the registration
	for _, cond := range options.conditions {
		if !r.validateCondition(cond) {
			return nil
		}
	}

	r.providers.Add(provider)

	return nil
}

func (r *Resolver) validateCondition(cond condition) bool {
	val, found, err := r.resolve(Request{
		unitaryTyp: StringType,
		query: queryByName{
			name: Name{
				name: cond.namedStringComponent,
				typ:  StringType,
			},
		},
		validator: validatorUniqueOptional{},
		collector: collectorUnique{},
	})
	if err != nil || !found {
		return false
	}

	return cond.operator(val.String(), cond.value)
}

func tryGetAt[T any](slice []T, index int) (val T, found bool) {
	if index < 0 || index >= len(slice) {
		return val, false
	}
	return slice[index], true
}

func (r *Resolver) MustRegister(reg Registrable, opts ...option.Option[RegistrableOptions]) *Resolver {
	err := r.Register(reg, opts...)
	if err != nil {
		panic(fmt.Sprintf("failed to register provider %T:\n\t%v", reg, err))
	}
	return r
}

func (r *Resolver) Close() error {
	// close all the stored components
	return r.store.Close()
}

// Resolve attempts to resolve a component of type T from the resolver.
func Resolve[T any](resolver *Resolver) (T, error) {
	var zero T
	lookFor := reflect.TypeOf((*T)(nil)).Elem()
	if lookFor == nil {
		return zero, fmt.Errorf("type %T is not a valid type", zero)
	}

	val, _, err := resolveTyped[T](
		resolver,
		Request{
			unitaryTyp: lookFor,
			query:      queryByType{typ: lookFor},
			validator:  validatorUniqueMandatory{},
			collector:  collectorUnique{},
		},
	)
	return val, err
}

// ResolveNamed attempts to resolve a named component of type T from the resolver.
func ResolveNamed[T any](resolver *Resolver, name string) (T, error) {
	var zero T
	lookFor := reflect.TypeOf((*T)(nil)).Elem()
	if lookFor == nil {
		return zero, fmt.Errorf("type %T is not a valid type", zero)
	}

	val, _, err := resolveTyped[T](
		resolver,
		Request{
			unitaryTyp: lookFor,
			query: queryByName{
				name: Name{name: name, typ: lookFor},
			},
			validator: validatorUniqueMandatory{},
			collector: collectorUnique{},
		},
	)
	return val, err
}

// ResolveAll attempts to resolve all components of type T from the resolver.
func ResolveAll[T any](resolver *Resolver) ([]T, error) {
	lookFor := reflect.TypeOf((*T)(nil)).Elem()

	val, _, err := resolveTyped[[]T](
		resolver,
		Request{
			unitaryTyp: lookFor,
			query:      queryByType{typ: lookFor},
			validator:  validatorMultiple{},
			collector:  collectorMultipleAsSlice{},
		},
	)
	return val, err
}

// TryResolve attempts to resolve a component of type T from the resolver.
//
// It returns the resolved value, a boolean indicating if it was found, and an error if any occurred during resolution.
func TryResolve[T any](resolver *Resolver) (value T, found bool, err error) {
	var zero T
	lookFor := reflect.TypeOf((*T)(nil)).Elem()
	if lookFor == nil {
		return zero, false, fmt.Errorf("type %T is not a valid type", zero)
	}

	return resolveTyped[T](
		resolver,
		Request{
			unitaryTyp: lookFor,
			query:      queryByType{typ: lookFor},
			validator:  validatorUniqueOptional{},
			collector:  collectorUnique{},
		},
	)
}

// TryResolveNamed attempts to resolve a component of name n from the resolver.
//
// It returns the resolved value, a boolean indicating if it was found, and an error if any occurred during resolution.
func TryResolveNamed[T any](resolver *Resolver, name string) (value T, found bool, err error) {
	var zero T
	lookFor := reflect.TypeOf((*T)(nil)).Elem()
	if lookFor == nil {
		return zero, false, fmt.Errorf("type %T is not a valid type", zero)
	}

	return resolveTyped[T](
		resolver,
		Request{
			unitaryTyp: lookFor,
			query: queryByName{
				name: Name{name: name, typ: lookFor},
			},
			validator: validatorUniqueOptional{},
			collector: collectorUnique{},
		},
	)
}

func resolveTyped[T any](resolver *Resolver, req Request) (val T, found bool, err error) {
	resolved, found, err := resolver.resolve(req)
	if err != nil {
		return val, false, fmt.Errorf("failed to resolve request %s:\n\t%w", req, err)
	}
	if !found {
		return val, false, nil
	}
	val, err = unReflect[T](resolved)
	return val, true, err
}

func (r *Resolver) resolve(req Request) (val reflect.Value, found bool, err error) {
	if perfOutput {
		start := time.Now()
		defer func() {
			fmt.Printf("resolved %s in %s\n", req, time.Since(start))
		}()
	}

	if req.tracker == nil {
		req.tracker = NewTracker()
	}

	results, err := req.query.find(r)
	if err != nil {
		return reflect.Value{}, false, fmt.Errorf("failed to resolve provider(s) from request %v:\n\t%w", req, err)
	}
	err = req.validator.validate(results)
	if err != nil {
		return reflect.Value{}, false, fmt.Errorf("failed to validate results for request %v:\n\t%w", req, err)
	}
	return req.collector.collect(req.unitaryTyp, r, results, req.tracker)
}

func compareByPriority(p1, p2 Provider) fn.ComparisonResult {
	if p1.Priority() < p2.Priority() {
		return fn.Less
	}
	if p1.Priority() > p2.Priority() {
		return fn.Greater
	}
	return fn.Equal
}

func unReflect[T any](v reflect.Value) (res T, err error) {
	res, ok := v.Interface().(T)
	if !ok {
		return res, fmt.Errorf("value %v is not of type %T", v, res)
	}
	return res, nil
}

func (r *Resolver) Describe() string {
	var b strings.Builder
	b.WriteString("* Providers:\n")
	for _, p := range r.providers.All() {
		providerStr := ""
		if reflect.TypeOf(p).Implements(StringerType) {
			providerStr = p.(fmt.Stringer).String()
		} else {
			providerStr = fmt.Sprintf("%T", p)
		}

		b.WriteString(fmt.Sprintf("\t- %s (priority=%d)\n", providerStr, p.Priority()))
		if desc := p.Description(); desc != "" {
			b.WriteString(fmt.Sprintf("\t\tdescription: %s\n", desc))
		}
		b.WriteString("\t\tprovides:\n")
		for _, n := range p.ListProvidableNames() {
			b.WriteString(fmt.Sprintf("\t\t\t- %s\n", n))
		}
		b.WriteString("\t\tdependencies:\n")
		for _, d := range p.Dependencies() {
			b.WriteString(fmt.Sprintf("\t\t\t- %s\n", d))
		}
	}
	b.WriteString("* Stored components:\n")
	for _, n := range r.store.ListNames() {
		comp, _ := r.store.Get(n)
		b.WriteString(fmt.Sprintf("\t- %s: %v\n", n, comp))
	}
	return b.String()
}
