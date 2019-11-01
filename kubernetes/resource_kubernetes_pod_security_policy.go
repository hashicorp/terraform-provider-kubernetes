package kubernetes

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	api "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	pkgApi "k8s.io/apimachinery/pkg/types"
)

// Use generated swagger docs from kubernetes' client-go to avoid copy/pasting them here
var (
	pspSpecDoc                                = api.PodSecurityPolicy{}.SwaggerDoc()["spec"]
	pspSpecAllowPrivilegeEscalationDoc        = api.PodSecurityPolicySpec{}.SwaggerDoc()["allowPrivilegeEscalation"]
	pspSpecAllowedCapabilitiesDoc             = api.PodSecurityPolicySpec{}.SwaggerDoc()["allowedCapabilities"]
	pspSpecAllowedFlexVolumesDoc              = api.PodSecurityPolicySpec{}.SwaggerDoc()["allowedFlexVolumes"]
	pspAllowedFlexVolumesDriverDoc            = api.AllowedFlexVolume{}.SwaggerDoc()["driver"]
	pspSpecAllowedHostPathsDoc                = api.PodSecurityPolicySpec{}.SwaggerDoc()["allowedHostPaths"]
	pspAllowedHostPathsPathPrefixDoc          = api.AllowedHostPath{}.SwaggerDoc()["pathPrefix"]
	pspAllowedHostPathsReadOnlyDoc            = api.AllowedHostPath{}.SwaggerDoc()["readOnly"]
	pspSpecAllowedProcMountTypesDoc           = api.PodSecurityPolicySpec{}.SwaggerDoc()["allowedProcMountTypes"]
	pspSpecAllowedUnsafeSysctlsDoc            = api.PodSecurityPolicySpec{}.SwaggerDoc()["allowedUnsafeSysctls"]
	pspSpecDefaultAddCapabilitiesDoc          = api.PodSecurityPolicySpec{}.SwaggerDoc()["defaultAddCapabilities"]
	pspSpecDefaultAllowPrivilegeEscalationDoc = api.PodSecurityPolicySpec{}.SwaggerDoc()["defaultAllowPrivilegeEscalation"]
	pspSpecForbiddenSysctlsDoc                = api.PodSecurityPolicySpec{}.SwaggerDoc()["forbiddenSysctls"]
	pspSpecFSGroupDoc                         = api.PodSecurityPolicySpec{}.SwaggerDoc()["fsGroup"]
	pspFSGroupIDRangeDoc                      = api.FSGroupStrategyOptions{}.SwaggerDoc()["ranges"]
	pspIDRangeMinDoc                          = api.IDRange{}.SwaggerDoc()["min"]
	pspIDRangeMaxDoc                          = api.IDRange{}.SwaggerDoc()["max"]
	pspFSGroupRuleDoc                         = api.FSGroupStrategyOptions{}.SwaggerDoc()["rule"]
	pspSpecHostIPCDoc                         = api.PodSecurityPolicySpec{}.SwaggerDoc()["hostIPC"]
	pspSpecHostNetworkDoc                     = api.PodSecurityPolicySpec{}.SwaggerDoc()["hostNetwork"]
	pspSpecHostPIDDoc                         = api.PodSecurityPolicySpec{}.SwaggerDoc()["hostPID"]
	pspSpecHostPortsDoc                       = api.PodSecurityPolicySpec{}.SwaggerDoc()["hostPorts"]
	pspHostPortRangeMinDoc                    = api.HostPortRange{}.SwaggerDoc()["min"]
	pspHostPortRangeMaxDoc                    = api.HostPortRange{}.SwaggerDoc()["max"]
	pspSpecPrivilegedDoc                      = api.PodSecurityPolicySpec{}.SwaggerDoc()["privileged"]
	pspSpecReadOnlyRootFilesystemDoc          = api.PodSecurityPolicySpec{}.SwaggerDoc()["readOnlyRootFilesystem"]
	pspSpecRequiredDropCapabilitiesDoc        = api.PodSecurityPolicySpec{}.SwaggerDoc()["requiredDropCapabilities"]
	pspSpecRunAsUserDoc                       = api.PodSecurityPolicySpec{}.SwaggerDoc()["runAsUser"]
	pspRunAsUserIDRangeDoc                    = api.RunAsUserStrategyOptions{}.SwaggerDoc()["ranges"]
	pspRunAsUserRuleDoc                       = api.RunAsUserStrategyOptions{}.SwaggerDoc()["rule"]
	pspSpecSELinuxDoc                         = api.PodSecurityPolicySpec{}.SwaggerDoc()["seLinux"]
	pspSELinuxOptionsDoc                      = api.SELinuxStrategyOptions{}.SwaggerDoc()["seLinuxOptions"]
	pspSELinuxOptionsLevelDoc                 = api.SELinuxStrategyOptions{}.SwaggerDoc()["level"]
	pspSELinuxOptionsRoleDoc                  = api.SELinuxStrategyOptions{}.SwaggerDoc()["role"]
	pspSELinuxOptionsTypeDoc                  = api.SELinuxStrategyOptions{}.SwaggerDoc()["type"]
	pspSELinuxOptionsUserDoc                  = api.SELinuxStrategyOptions{}.SwaggerDoc()["user"]
	pspSELinuxOptionsRuleDoc                  = api.SELinuxStrategyOptions{}.SwaggerDoc()["rule"]
	pspSpecSupplementalGroupsDoc              = api.PodSecurityPolicySpec{}.SwaggerDoc()["supplementalGroups"]
	pspSupplementalGroupsRangesDoc            = api.SupplementalGroupsStrategyOptions{}.SwaggerDoc()["ranges"]
	pspSupplementalGroupsRuleDoc              = api.SupplementalGroupsStrategyOptions{}.SwaggerDoc()["rule"]
	pspSpecVolumesDoc                         = api.PodSecurityPolicySpec{}.SwaggerDoc()["volumes"]
	pspSpecRunAsGroupDoc                      = api.PodSecurityPolicySpec{}.SwaggerDoc()["runAsGroup"]
	pspRunAsGroupIDRangeDoc                   = api.RunAsGroupStrategyOptions{}.SwaggerDoc()["ranges"]
	pspRunAsGroupRuleDoc                      = api.RunAsGroupStrategyOptions{}.SwaggerDoc()["rule"]
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
	conn := meta.(*KubeClientsets).MainClientset

	metadata := expandMetadata(d.Get("metadata").([]interface{}))
	spec, err := expandPodSecurityPolicySpec(d.Get("spec").([]interface{}))

	if err != nil {
		return err
	}

	psp := &api.PodSecurityPolicy{
		ObjectMeta: metadata,
		Spec:       spec,
	}

	log.Printf("[INFO] Creating new PodSecurityPolicy: %#v", psp)
	out, err := conn.ExtensionsV1beta1().PodSecurityPolicies().Create(psp)

	if err != nil {
		return err
	}
	log.Printf("[INFO] Submitted new PodSecurityPolicy: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesPodSecurityPolicyRead(d, meta)
}

func resourceKubernetesPodSecurityPolicyRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	name := d.Id()

	log.Printf("[INFO] Reading PodSecurityPolicy %s", name)
	psp, err := conn.ExtensionsV1beta1().PodSecurityPolicies().Get(name, meta_v1.GetOptions{})
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
	conn := meta.(*KubeClientsets).MainClientset

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
	out, err := conn.ExtensionsV1beta1().PodSecurityPolicies().Patch(name, pkgApi.JSONPatchType, data)
	if err != nil {
		return fmt.Errorf("Failed to update PodSecurityPolicy: %s", err)
	}
	log.Printf("[INFO] Submitted updated PodSecurityPolicy: %#v", out)
	d.SetId(out.Name)

	return resourceKubernetesPodSecurityPolicyRead(d, meta)
}

func resourceKubernetesPodSecurityPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*KubeClientsets).MainClientset

	name := d.Id()

	log.Printf("[INFO] Deleting PodSecurityPolicy: %#v", name)
	err := conn.ExtensionsV1beta1().PodSecurityPolicies().Delete(name, &meta_v1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Printf("[INFO] PodSecurityPolicy %s deleted", name)

	return nil
}

func resourceKubernetesPodSecurityPolicyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	conn := meta.(*KubeClientsets).MainClientset

	name := d.Id()

	log.Printf("[INFO] Checking PodSecurityPolicy %s", name)
	_, err := conn.ExtensionsV1beta1().PodSecurityPolicies().Get(name, meta_v1.GetOptions{})
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
