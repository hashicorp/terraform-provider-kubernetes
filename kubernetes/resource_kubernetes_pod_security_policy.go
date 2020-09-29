package kubernetes

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	policy "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

// Use generated swagger docs from kubernetes' client-go to avoid copy/pasting them here
var (
	pspSpecDoc                                = policy.PodSecurityPolicy{}.SwaggerDoc()["spec"]
	pspSpecAllowPrivilegeEscalationDoc        = policy.PodSecurityPolicySpec{}.SwaggerDoc()["allowPrivilegeEscalation"]
	pspSpecAllowedCapabilitiesDoc             = policy.PodSecurityPolicySpec{}.SwaggerDoc()["allowedCapabilities"]
	pspSpecAllowedFlexVolumesDoc              = policy.PodSecurityPolicySpec{}.SwaggerDoc()["allowedFlexVolumes"]
	pspAllowedFlexVolumesDriverDoc            = policy.AllowedFlexVolume{}.SwaggerDoc()["driver"]
	pspSpecAllowedHostPathsDoc                = policy.PodSecurityPolicySpec{}.SwaggerDoc()["allowedHostPaths"]
	pspAllowedHostPathsPathPrefixDoc          = policy.AllowedHostPath{}.SwaggerDoc()["pathPrefix"]
	pspAllowedHostPathsReadOnlyDoc            = policy.AllowedHostPath{}.SwaggerDoc()["readOnly"]
	pspSpecAllowedProcMountTypesDoc           = policy.PodSecurityPolicySpec{}.SwaggerDoc()["allowedProcMountTypes"]
	pspSpecAllowedUnsafeSysctlsDoc            = policy.PodSecurityPolicySpec{}.SwaggerDoc()["allowedUnsafeSysctls"]
	pspSpecDefaultAddCapabilitiesDoc          = policy.PodSecurityPolicySpec{}.SwaggerDoc()["defaultAddCapabilities"]
	pspSpecDefaultAllowPrivilegeEscalationDoc = policy.PodSecurityPolicySpec{}.SwaggerDoc()["defaultAllowPrivilegeEscalation"]
	pspSpecForbiddenSysctlsDoc                = policy.PodSecurityPolicySpec{}.SwaggerDoc()["forbiddenSysctls"]
	pspSpecFSGroupDoc                         = policy.PodSecurityPolicySpec{}.SwaggerDoc()["fsGroup"]
	pspFSGroupIDRangeDoc                      = policy.FSGroupStrategyOptions{}.SwaggerDoc()["ranges"]
	pspIDRangeMinDoc                          = policy.IDRange{}.SwaggerDoc()["min"]
	pspIDRangeMaxDoc                          = policy.IDRange{}.SwaggerDoc()["max"]
	pspFSGroupRuleDoc                         = policy.FSGroupStrategyOptions{}.SwaggerDoc()["rule"]
	pspSpecHostIPCDoc                         = policy.PodSecurityPolicySpec{}.SwaggerDoc()["hostIPC"]
	pspSpecHostNetworkDoc                     = policy.PodSecurityPolicySpec{}.SwaggerDoc()["hostNetwork"]
	pspSpecHostPIDDoc                         = policy.PodSecurityPolicySpec{}.SwaggerDoc()["hostPID"]
	pspSpecHostPortsDoc                       = policy.PodSecurityPolicySpec{}.SwaggerDoc()["hostPorts"]
	pspHostPortRangeMinDoc                    = policy.HostPortRange{}.SwaggerDoc()["min"]
	pspHostPortRangeMaxDoc                    = policy.HostPortRange{}.SwaggerDoc()["max"]
	pspSpecPrivilegedDoc                      = policy.PodSecurityPolicySpec{}.SwaggerDoc()["privileged"]
	pspSpecReadOnlyRootFilesystemDoc          = policy.PodSecurityPolicySpec{}.SwaggerDoc()["readOnlyRootFilesystem"]
	pspSpecRequiredDropCapabilitiesDoc        = policy.PodSecurityPolicySpec{}.SwaggerDoc()["requiredDropCapabilities"]
	pspSpecRunAsUserDoc                       = policy.PodSecurityPolicySpec{}.SwaggerDoc()["runAsUser"]
	pspRunAsUserIDRangeDoc                    = policy.RunAsUserStrategyOptions{}.SwaggerDoc()["ranges"]
	pspRunAsUserRuleDoc                       = policy.RunAsUserStrategyOptions{}.SwaggerDoc()["rule"]
	pspSpecSELinuxDoc                         = policy.PodSecurityPolicySpec{}.SwaggerDoc()["seLinux"]
	pspSELinuxOptionsDoc                      = policy.SELinuxStrategyOptions{}.SwaggerDoc()["seLinuxOptions"]
	pspSELinuxOptionsLevelDoc                 = policy.SELinuxStrategyOptions{}.SwaggerDoc()["level"]
	pspSELinuxOptionsRoleDoc                  = policy.SELinuxStrategyOptions{}.SwaggerDoc()["role"]
	pspSELinuxOptionsTypeDoc                  = policy.SELinuxStrategyOptions{}.SwaggerDoc()["type"]
	pspSELinuxOptionsUserDoc                  = policy.SELinuxStrategyOptions{}.SwaggerDoc()["user"]
	pspSELinuxOptionsRuleDoc                  = policy.SELinuxStrategyOptions{}.SwaggerDoc()["rule"]
	pspSpecSupplementalGroupsDoc              = policy.PodSecurityPolicySpec{}.SwaggerDoc()["supplementalGroups"]
	pspSupplementalGroupsRangesDoc            = policy.SupplementalGroupsStrategyOptions{}.SwaggerDoc()["ranges"]
	pspSupplementalGroupsRuleDoc              = policy.SupplementalGroupsStrategyOptions{}.SwaggerDoc()["rule"]
	pspSpecVolumesDoc                         = policy.PodSecurityPolicySpec{}.SwaggerDoc()["volumes"]
	pspSpecRunAsGroupDoc                      = policy.PodSecurityPolicySpec{}.SwaggerDoc()["runAsGroup"]
	pspRunAsGroupIDRangeDoc                   = policy.RunAsGroupStrategyOptions{}.SwaggerDoc()["ranges"]
	pspRunAsGroupRuleDoc                      = policy.RunAsGroupStrategyOptions{}.SwaggerDoc()["rule"]
)

func resourceKubernetesPodSecurityPolicy() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPodSecurityPolicyCreate,
		Read:   resourceKubernetesPodSecurityPolicyRead,
		Exists: resourceKubernetesPodSecurityPolicyExists,
		Update: resourceKubernetesPodSecurityPolicyUpdate,
		Delete: resourceKubernetesPodSecurityPolicyDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"metadata": metadataSchema("podsecuritypolicy", false),
			"spec": {
				Type:        schema.TypeList,
				Description: pspSpecDoc,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allow_privilege_escalation": {
							Type:        schema.TypeBool,
							Description: pspSpecAllowPrivilegeEscalationDoc,
							Optional:    true,
							Computed:    true,
						},
						"allowed_capabilities": {
							Type:        schema.TypeList,
							Description: pspSpecAllowedCapabilitiesDoc,
							Optional:    true,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"allowed_flex_volumes": {
							Type:        schema.TypeList,
							Description: pspSpecAllowedFlexVolumesDoc,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"driver": {
										Type:        schema.TypeString,
										Description: pspAllowedFlexVolumesDriverDoc,
										Required:    true,
									},
								},
							},
						},
						"allowed_host_paths": {
							Type:        schema.TypeList,
							Description: pspSpecAllowedHostPathsDoc,
							Optional:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"path_prefix": {
										Type:        schema.TypeString,
										Description: pspAllowedHostPathsPathPrefixDoc,
										Required:    true,
									},
									"read_only": {
										Type:        schema.TypeBool,
										Description: pspAllowedHostPathsReadOnlyDoc,
										Optional:    true,
									},
								},
							},
						},
						"allowed_proc_mount_types": {
							Type:        schema.TypeList,
							Description: pspSpecAllowedProcMountTypesDoc,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"allowed_unsafe_sysctls": {
							Type:        schema.TypeList,
							Description: pspSpecAllowedUnsafeSysctlsDoc,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"default_add_capabilities": {
							Type:        schema.TypeList,
							Description: pspSpecDefaultAddCapabilitiesDoc,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"default_allow_privilege_escalation": {
							Type:        schema.TypeBool,
							Description: pspSpecDefaultAllowPrivilegeEscalationDoc,
							Optional:    true,
							Computed:    true,
						},
						"forbidden_sysctls": {
							Type:        schema.TypeList,
							Description: pspSpecForbiddenSysctlsDoc,
							Optional:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"fs_group": {
							Type:        schema.TypeList,
							Description: pspSpecFSGroupDoc,
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"range": {
										Type:        schema.TypeList,
										Description: pspFSGroupIDRangeDoc,
										Optional:    true,
										Computed:    true,
										Elem: &schema.Resource{
											Schema: idRangeSchema(),
										},
									},
									"rule": {
										Type:        schema.TypeString,
										Description: pspFSGroupRuleDoc,
										Required:    true,
									},
								},
							},
						},
						"host_ipc": {
							Type:        schema.TypeBool,
							Description: pspSpecHostIPCDoc,
							Optional:    true,
							Computed:    true,
						},
						"host_network": {
							Type:        schema.TypeBool,
							Description: pspSpecHostNetworkDoc,
							Optional:    true,
							Computed:    true,
						},
						"host_pid": {
							Type:        schema.TypeBool,
							Description: pspSpecHostPIDDoc,
							Optional:    true,
							Computed:    true,
						},
						"host_ports": {
							Type:        schema.TypeList,
							Description: pspSpecHostPortsDoc,
							Optional:    true,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"min": {
										Type:        schema.TypeInt,
										Description: pspHostPortRangeMinDoc,
										Required:    true,
									},
									"max": {
										Type:        schema.TypeInt,
										Description: pspHostPortRangeMaxDoc,
										Required:    true,
									},
								},
							},
						},
						"privileged": {
							Type:        schema.TypeBool,
							Description: pspSpecPrivilegedDoc,
							Optional:    true,
							Computed:    true,
						},
						"read_only_root_filesystem": {
							Type:        schema.TypeBool,
							Description: pspSpecReadOnlyRootFilesystemDoc,
							Optional:    true,
							Computed:    true,
						},
						"required_drop_capabilities": {
							Type:        schema.TypeList,
							Description: pspSpecRequiredDropCapabilitiesDoc,
							Optional:    true,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
						"run_as_user": {
							Type:        schema.TypeList,
							Description: pspSpecRunAsUserDoc,
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"range": {
										Type:        schema.TypeList,
										Description: pspRunAsUserIDRangeDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: idRangeSchema(),
										},
									},
									"rule": {
										Type:        schema.TypeString,
										Description: pspRunAsUserRuleDoc,
										Required:    true,
									},
								},
							},
						},
						"run_as_group": {
							Type:        schema.TypeList,
							Description: pspSpecRunAsGroupDoc,
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"range": {
										Type:        schema.TypeList,
										Description: pspRunAsGroupIDRangeDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: idRangeSchema(),
										},
									},
									"rule": {
										Type:        schema.TypeString,
										Description: pspRunAsGroupRuleDoc,
										Required:    true,
									},
								},
							},
						},
						"se_linux": {
							Type:        schema.TypeList,
							Description: pspSpecSELinuxDoc,
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"se_linux_options": {
										Type:        schema.TypeList,
										Description: pspSELinuxOptionsDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"level": {
													Type:        schema.TypeString,
													Description: pspSELinuxOptionsLevelDoc,
													Required:    true,
												},
												"role": {
													Type:        schema.TypeString,
													Description: pspSELinuxOptionsRoleDoc,
													Required:    true,
												},
												"type": {
													Type:        schema.TypeString,
													Description: pspSELinuxOptionsTypeDoc,
													Required:    true,
												},
												"user": {
													Type:        schema.TypeString,
													Description: pspSELinuxOptionsUserDoc,
													Required:    true,
												},
											},
										},
									},
									"rule": {
										Type:        schema.TypeString,
										Description: pspSELinuxOptionsRuleDoc,
										Required:    true,
									},
								},
							},
						},
						"supplemental_groups": {
							Type:        schema.TypeList,
							Description: pspSpecSupplementalGroupsDoc,
							Required:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"range": {
										Type:        schema.TypeList,
										Description: pspSupplementalGroupsRangesDoc,
										Optional:    true,
										Elem: &schema.Resource{
											Schema: idRangeSchema(),
										},
									},
									"rule": {
										Type:        schema.TypeString,
										Description: pspSupplementalGroupsRuleDoc,
										Required:    true,
									},
								},
							},
						},
						"volumes": {
							Type:        schema.TypeList,
							Description: pspSpecVolumesDoc,
							Optional:    true,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceKubernetesPodSecurityPolicyCreate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodSecurityPolicySpec(d.Get("spec").([]interface{}))

	if err != nil {
		return err
	}

	psp := &policy.PodSecurityPolicy{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new PodSecurityPolicy: %#v", psp)
	out, err := conn.PolicyV1beta1().PodSecurityPolicies().Create(ctx, psp, metav1.CreateOptions{})

	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new PodSecurityPolicy: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesPodSecurityPolicyRead(d, meta)
}

func resourceKubernetesPodSecurityPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()

	log.Printf("[INFO] Reading PodSecurityPolicy %s", name)
	psp, err := conn.PolicyV1beta1().PodSecurityPolicies().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log.Printf("[DEBUG] Received error: %#v", err)
		return err
	}

	log.Printf("[INFO] Received PodSecurityPolicy: %#v", psp)
	err = d.Set("metadata", flattenMetadata(psp.ObjectMeta, d))
	if err != nil {
		return err
	}

	flattenedSpec := flattenPodSecurityPolicySpec(psp.Spec)
	log.Printf("[DEBUG] Flattened PodSecurityPolicy roleRef: %#v", flattenedSpec)
	err = d.Set("spec", flattenedSpec)
	if err != nil {
		return err
	}

	return nil
}

func resourceKubernetesPodSecurityPolicyUpdate(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()

	ops := patchMetadata("metadata.0.", "/metadata/", d)

	if d.HasChange("spec") {
		diffOps, err := patchPodSecurityPolicySpec("spec.0.", "/spec", d)
		if err != nil {
			return err
		}
		ops = append(ops, *diffOps...)
	}
	data, err := ops.MarshalJSON()
	if err != nil {
		return fmt.Errorf("Failed to marshal update operations: %s", err)
	}
	log.Printf("[INFO] Updating PodSecurityPolicy %q: %v", name, string(data))
	out, err := conn.PolicyV1beta1().PodSecurityPolicies().Patch(ctx, name, pkgApi.JSONPatchType, data, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("Failed to update PodSecurityPolicy: %s", err)
	}
	log.Printf("[INFO] Submitted updated PodSecurityPolicy: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesPodSecurityPolicyRead(d, meta)
}

func resourceKubernetesPodSecurityPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return err
	}
	ctx := context.TODO()

	name := d.Id()

	log.Printf("[INFO] Deleting PodSecurityPolicy: %#v", name)
	err = conn.PolicyV1beta1().PodSecurityPolicies().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] PodSecurityPolicy %s deleted", name)

	return nil
}

func resourceKubernetesPodSecurityPolicyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn, err := meta.(KubeClientsets).MainClientset()
	if err != nil {
		return false, err
	}
	ctx := context.TODO()

	name := d.Id()

	log.Printf("[INFO] Checking PodSecurityPolicy %s", name)
	_, err = conn.PolicyV1beta1().PodSecurityPolicies().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if statusErr, ok := err.(*errors.StatusError); ok && statusErr.ErrStatus.Code == 404 {
			return false, nil
		}
		log.Printf("[DEBUG] Received error: %#v", err)
	}
	return true, err
}

func idRangeSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"min": {
			Type:        schema.TypeInt,
			Description: pspIDRangeMinDoc,
			Required:    true,
		},
		"max": {
			Type:        schema.TypeInt,
			Description: pspIDRangeMaxDoc,
			Required:    true,
		},
	}
}
