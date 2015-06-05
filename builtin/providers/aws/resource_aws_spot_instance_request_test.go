package aws

import (
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSSpotInstanceRequest_basic(t *testing.T) {
	var sir ec2.SpotInstanceRequest

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAWSSpotInstanceRequestDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccAWSSpotInstanceRequestConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAWSSpotInstanceRequestExists("aws_spot_instance_request.foo", &sir),
					testAccCheckAWSSpotInstanceRequestAttributes(&sir),
				),
			},
		},
	})
}

func testAccCheckAWSSpotInstanceRequestDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*AWSClient).ec2conn

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_spot_instance_request" {
			continue
		}

		req := &ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIDs: []*string{aws.String(rs.Primary.ID)},
		}

		_, err := conn.DescribeSpotInstanceRequests(req)

		if err == nil {
			return fmt.Errorf("Spot instance should be gone, but it's still here.")
		}

		// Verify the error is an API error, not something else
		_, ok := err.(awserr.Error)
		if !ok {
			return err
		}
	}

	return nil
}

func testAccCheckAWSSpotInstanceRequestExists(
	n string, sir *ec2.SpotInstanceRequest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No SNS subscription with that ARN exists")
		}

		conn := testAccProvider.Meta().(*AWSClient).ec2conn

		params := &ec2.DescribeSpotInstanceRequestsInput{}
		resp, err := conn.DescribeSpotInstanceRequests(params)

		if err != nil {
			return err
		}

		if v := len(resp.SpotInstanceRequests); v != 1 {
			return fmt.Errorf("Expected 1 request returned, got %d", v)
		}

		*sir = *resp.SpotInstanceRequests[0]

		return nil
	}
}

func testAccCheckAWSSpotInstanceRequestAttributes(
	sir *ec2.SpotInstanceRequest) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if *sir.SpotPrice != "0.05" {
			return fmt.Errorf("Unexpected spot price: %s", *sir.SpotPrice)
		}
		return nil
	}
}

const testAccAWSSpotInstanceRequestConfig = `
resource "aws_spot_instance_request" "foo" {
	ami = "ami-4fccb37f"
	instance_type = "m1.small"
	// base price is $0.044 hourly, so bidding above that should theoretically
	// always fulfill
	spot_price = "0.05"
}
`
