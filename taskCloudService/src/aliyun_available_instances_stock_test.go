package main

import (
	"testing"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/dara"
)

func TestCollectAvailableInstanceTypeIDsSkipsNonAvailable(t *testing.T) {
	available := "Available"
	soldOut := "SoldOut"
	itAvailable := "ecs.available.large"
	itSoldOut := "ecs.soldout.large"
	resp := &ecsclient.DescribeAvailableResourceResponse{
		Body: &ecsclient.DescribeAvailableResourceResponseBody{
			AvailableZones: &ecsclient.DescribeAvailableResourceResponseBodyAvailableZones{
				AvailableZone: []*ecsclient.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZone{
					{
						AvailableResources: &ecsclient.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZoneAvailableResources{
							AvailableResource: []*ecsclient.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZoneAvailableResourcesAvailableResource{
								{
									SupportedResources: &ecsclient.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZoneAvailableResourcesAvailableResourceSupportedResources{
										SupportedResource: []*ecsclient.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZoneAvailableResourcesAvailableResourceSupportedResourcesSupportedResource{
											{Value: dara.String(itAvailable), Status: dara.String(available)},
											{Value: dara.String(itSoldOut), Status: dara.String(soldOut)},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	got := collectAvailableInstanceTypeIDs(resp)
	if len(got) != 1 || got[0] != itAvailable {
		t.Fatalf("expected only available instance type, got %#v", got)
	}
}
