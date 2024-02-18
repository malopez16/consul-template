// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package dependency

import (
	"fmt"
	"github.com/hashicorp/consul/proto-public/pbresource"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewKVListQuery(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  *KVListQuery
		err  bool
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *pbresource.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				tenancyHelper.AppendTenancyInfo("empty", tenancy),
				"",
				&KVListQuery{},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_only", tenancy),
				"@dc1",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("query_only", tenancy),
				fmt.Sprintf("?ns=%s", tenancy.Namespace),
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("invalid query param (unsupported key)", tenancy),
				"prefix?unsupported=foo",
				nil,
				true,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("prefix", tenancy),
				"prefix",
				&KVListQuery{
					prefix: "prefix",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc", tenancy),
				"prefix@dc1",
				&KVListQuery{
					prefix: "prefix",
					dc:     "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("partition", tenancy),
				fmt.Sprintf("prefix?partition=%s", tenancy.Partition),
				&KVListQuery{
					prefix:    "prefix",
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace", tenancy),
				fmt.Sprintf("prefix?ns=%s", tenancy.Namespace),
				&KVListQuery{
					prefix:    "prefix",
					namespace: tenancy.Namespace,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("namespace_and_partition", tenancy),
				fmt.Sprintf("prefix?ns=%s&partition=%s", tenancy.Namespace, tenancy.Partition),
				&KVListQuery{
					prefix:    "prefix",
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dc_namespace_and_partition", tenancy),
				fmt.Sprintf("prefix?ns=%s&partition=%s@dc1", tenancy.Namespace, tenancy.Partition),
				&KVListQuery{
					prefix:    "prefix",
					namespace: tenancy.Namespace,
					partition: tenancy.Partition,
					dc:        "dc1",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("empty_query", tenancy),
				"prefix?ns=&partition=",
				&KVListQuery{
					prefix:    "prefix",
					namespace: "",
					partition: "",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dots", tenancy),
				"prefix.with.dots",
				&KVListQuery{
					prefix: "prefix.with.dots",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("slashes", tenancy),
				"prefix/with/slashes",
				&KVListQuery{
					prefix: "prefix/with/slashes",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("dashes", tenancy),
				"prefix-with-dashes",
				&KVListQuery{
					prefix: "prefix-with-dashes",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("leading_slash", tenancy),
				"/leading/slash",
				&KVListQuery{
					prefix: "leading/slash",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("trailing_slash", tenancy),
				"trailing/slash/",
				&KVListQuery{
					prefix: "trailing/slash/",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("underscores", tenancy),
				"prefix_with_underscores",
				&KVListQuery{
					prefix: "prefix_with_underscores",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("special_characters", tenancy),
				"config/facet:größe-lf-si",
				&KVListQuery{
					prefix: "config/facet:größe-lf-si",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("splat", tenancy),
				"config/*/timeouts/",
				&KVListQuery{
					prefix: "config/*/timeouts/",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("slash", tenancy),
				"/",
				&KVListQuery{
					prefix: "/",
				},
				false,
			},
			testCase{
				tenancyHelper.AppendTenancyInfo("slash-slash", tenancy),
				"//",
				&KVListQuery{
					prefix: "/",
				},
				false,
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			act, err := NewKVListQuery(tc.i)
			if (err != nil) != tc.err {
				t.Fatal(err)
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestKVListQuery_Fetch(t *testing.T) {
	for _, tenancy := range tenancyHelper.TestTenancies() {
		testConsul.SetKVString(t, fmt.Sprintf("test-kv-list/prefix/foo?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace), fmt.Sprintf("bar-%s-%s", tenancy.Partition, tenancy.Namespace))
		testConsul.SetKVString(t, fmt.Sprintf("test-kv-list/prefix/zip?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace), fmt.Sprintf("zap-%s-%s", tenancy.Partition, tenancy.Namespace))
		testConsul.SetKVString(t, fmt.Sprintf("test-kv-list/prefix/wave/ocean?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace), fmt.Sprintf("sleek-%s-%s", tenancy.Partition, tenancy.Namespace))
	}

	type testCase struct {
		name string
		i    string
		exp  []*KeyPair
	}

	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *pbresource.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				"exists",
				fmt.Sprintf("test-kv-list/prefix?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				[]*KeyPair{
					{
						Path:  "test-kv-list/prefix/foo",
						Key:   "foo",
						Value: fmt.Sprintf("bar-%s-%s", tenancy.Partition, tenancy.Namespace),
					},
					{
						Path:  "test-kv-list/prefix/wave/ocean",
						Key:   "wave/ocean",
						Value: fmt.Sprintf("sleek-%s-%s", tenancy.Partition, tenancy.Namespace),
					},
					{
						Path:  "test-kv-list/prefix/zip",
						Key:   "zip",
						Value: fmt.Sprintf("zap-%s-%s", tenancy.Partition, tenancy.Namespace),
					},
				},
			},
			testCase{
				"trailing",
				fmt.Sprintf("test-kv-list/prefix/?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				[]*KeyPair{
					{
						Path:  "test-kv-list/prefix/foo",
						Key:   "foo",
						Value: fmt.Sprintf("bar-%s-%s", tenancy.Partition, tenancy.Namespace),
					},
					{
						Path:  "test-kv-list/prefix/wave/ocean",
						Key:   "wave/ocean",
						Value: fmt.Sprintf("sleek-%s-%s", tenancy.Partition, tenancy.Namespace),
					},
					{
						Path:  "test-kv-list/prefix/zip",
						Key:   "zip",
						Value: fmt.Sprintf("zap-%s-%s", tenancy.Partition, tenancy.Namespace),
					},
				},
			},
			testCase{
				"no_exist",
				fmt.Sprintf("test-kv-list/not/a/real/prefix/like/ever?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				[]*KeyPair{},
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVListQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}

			act, _, err := d.Fetch(testClients, nil)
			if err != nil {
				t.Fatal(err)
			}

			for _, p := range act.([]*KeyPair) {
				p.CreateIndex = 0
				p.ModifyIndex = 0
			}

			assert.Equal(t, tc.exp, act)
		})
	}

	tenancyHelper.RunWithTenancies(func(tenancy *pbresource.Tenancy) {
		d, err := NewKVListQuery(fmt.Sprintf("test-kv-list/prefix?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace))
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			for {
				data, _, err := d.Fetch(testClients, nil)
				if err != nil {
					errCh <- err
					return
				}
				dataCh <- data
			}
		}()

		select {
		case err := <-errCh:
			t.Fatal(err)
		case <-dataCh:
		}

		d.Stop()

		select {
		case err := <-errCh:
			if err != ErrStopped {
				t.Fatal(err)
			}
		case <-time.After(250 * time.Millisecond):
			t.Errorf("did not stop")
		}
	}, t, "stops")

	tenancyHelper.RunWithTenancies(func(tenancy *pbresource.Tenancy) {
		d, err := NewKVListQuery(fmt.Sprintf("test-kv-list/prefix?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace))
		if err != nil {
			t.Fatal(err)
		}

		_, qm, err := d.Fetch(testClients, nil)
		if err != nil {
			t.Fatal(err)
		}

		dataCh := make(chan interface{}, 1)
		errCh := make(chan error, 1)
		go func() {
			data, _, err := d.Fetch(testClients, &QueryOptions{WaitIndex: qm.LastIndex})
			if err != nil {
				errCh <- err
				return
			}
			dataCh <- data
		}()

		testConsul.SetKVString(t, fmt.Sprintf("test-kv-list/prefix/foo?partition=%s&namespace=%s", tenancy.Partition, tenancy.Namespace), "new-bar")

		select {
		case err := <-errCh:
			t.Fatal(err)
		case data := <-dataCh:
			typed := data.([]*KeyPair)
			if len(typed) == 0 {
				t.Fatal("bad length")
			}

			act := typed[0]
			act.CreateIndex = 0
			act.ModifyIndex = 0

			exp := &KeyPair{
				Path:  "test-kv-list/prefix/foo",
				Key:   "foo",
				Value: "new-bar",
			}

			assert.Equal(t, exp, act)
		}
	}, t, "fires_changes")
}

func TestKVListQuery_String(t *testing.T) {
	type testCase struct {
		name string
		i    string
		exp  string
	}
	cases := tenancyHelper.GenerateTenancyTests(func(tenancy *pbresource.Tenancy) []interface{} {
		return []interface{}{
			testCase{
				"prefix",
				"prefix",
				"kv.list(prefix)",
			},
			testCase{
				"dc",
				"prefix@dc1",
				"kv.list(prefix@dc1)",
			},
			testCase{
				"dc_partition",
				fmt.Sprintf("prefix?partition=%s@dc1", tenancy.Partition),
				fmt.Sprintf("kv.list(prefix@dc1@partition=%s)", tenancy.Partition),
			},
			testCase{
				"dc_namespace",
				fmt.Sprintf("prefix?ns=%s@dc1", tenancy.Namespace),
				fmt.Sprintf("kv.list(prefix@dc1@ns=%s)", tenancy.Namespace),
			},
			testCase{
				"dc_partition_namespace",
				fmt.Sprintf("prefix?partition=%s&ns=%s@dc1", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("kv.list(prefix@dc1@partition=%s@ns=%s)", tenancy.Partition, tenancy.Namespace),
			},
			testCase{
				"partition_namespace",
				fmt.Sprintf("prefix?partition=%s&ns=%s", tenancy.Partition, tenancy.Namespace),
				fmt.Sprintf("kv.list(prefix@partition=%s@ns=%s)", tenancy.Partition, tenancy.Namespace),
			},
		}
	})

	for i, test := range cases {
		tc := test.(testCase)
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			d, err := NewKVListQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
