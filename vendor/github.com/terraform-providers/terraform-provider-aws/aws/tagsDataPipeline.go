package aws

import (
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/datapipeline"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

// setTags is a helper to set the tags for a resource. It expects the
// tags field to be named "tags"
func setTagsDataPipeline(conn *datapipeline.DataPipeline, d *schema.ResourceData) error {
	if d.HasChange("tags") {
		oraw, nraw := d.GetChange("tags")
		o := oraw.(map[string]interface{})
		n := nraw.(map[string]interface{})
		create, remove := diffTagsDataPipeline(tagsFromMapDataPipeline(o), tagsFromMapDataPipeline(n))
		// Set tags
		if len(remove) > 0 {
			log.Printf("[DEBUG] Removing tags: %#v", remove)
			k := make([]*string, len(remove))
			for i, t := range remove {
				k[i] = t.Key
			}
			_, err := conn.RemoveTags(&datapipeline.RemoveTagsInput{
				PipelineId: aws.String(d.Id()),
				TagKeys:    k,
			})
			if err != nil {
				return err
			}
		}
		if len(create) > 0 {
			log.Printf("[DEBUG] Creating tags: %#v", create)
			_, err := conn.AddTags(&datapipeline.AddTagsInput{
				PipelineId: aws.String(d.Id()),
				Tags:       create,
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// diffTags takes our tags locally and the ones remotely and returns
// the set of tags that must be created, and the set of tags that must
// be destroyed.
func diffTagsDataPipeline(oldTags, newTags []*datapipeline.Tag) ([]*datapipeline.Tag, []*datapipeline.Tag) {
	// First, we're creating everything we have
	create := make(map[string]interface{})
	for _, t := range newTags {
		create[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}
	// Build the list of what to remove
	var remove []*datapipeline.Tag
	for _, t := range oldTags {
		old, ok := create[aws.StringValue(t.Key)]
		if !ok || old != aws.StringValue(t.Value) {
			// Delete it!
			remove = append(remove, t)
		} else if ok {
			// already present so remove from new
			delete(create, aws.StringValue(t.Key))
		}
	}
	return tagsFromMapDataPipeline(create), remove
}

// tagsFromMap returns the tags for the given map of data.
func tagsFromMapDataPipeline(m map[string]interface{}) []*datapipeline.Tag {
	var result []*datapipeline.Tag
	for k, v := range m {
		t := &datapipeline.Tag{
			Key:   aws.String(k),
			Value: aws.String(v.(string)),
		}
		if !tagIgnoredDataPipeline(t) {
			result = append(result, t)
		}
	}
	return result
}

// tagsToMap turns the list of tags into a map.
func tagsToMapDataPipeline(ts []*datapipeline.Tag) map[string]string {
	result := make(map[string]string)
	for _, t := range ts {
		if !tagIgnoredDataPipeline(t) {
			result[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
		}
	}
	return result
}

// compare a tag against a list of strings and checks if it should
// be ignored or not
func tagIgnoredDataPipeline(t *datapipeline.Tag) bool {
	filter := []string{"^aws:"}
	for _, v := range filter {
		log.Printf("[DEBUG] Matching %v with %v\n", v, aws.StringValue(t.Key))
		if r, _ := regexp.MatchString(v, aws.StringValue(t.Key)); r {
			log.Printf("[DEBUG] Found AWS specific tag %s (val: %s), ignoring.\n", aws.StringValue(t.Key), aws.StringValue(t.Value))
			return true
		}
	}
	return false
}
