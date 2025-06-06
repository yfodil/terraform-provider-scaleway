---
subcategory: "Domains and DNS"  
page_title: "Scaleway: scaleway_domain_registration"
---

# Resource: scaleway_domain_registration

The `scaleway_domain_registration` resource allows you to purchase and manage domain registrations with Scaleway. Using this resource you can register one or more domains for a specified duration, configure auto-renewal and DNSSEC options, and set contact information. You can supply an owner contact either by providing an existing contact ID or by specifying the complete contact details. The resource automatically returns additional contact information (administrative and technical) as provided by the Scaleway API.

Refer to the [Domains and DNS documentation](https://www.scaleway.com/en/docs/network/domains-and-dns/) and the [API documentation](https://developers.scaleway.com/) for more details.

## Example Usage

### Purchase a Single Domain

The following example purchases a domain with a one-year registration period and specifies the owner contact details. Administrative and technical contacts are returned by the API.

```terraform
resource "scaleway_domain_registration" "test" {
  domain_names      = ["example.com"]
  duration_in_years = 1

  owner_contact {
    legal_form                  = "individual"
    firstname                   = "John"
    lastname                    = "DOE"
    email                       = "john.doe@example.com"
    phone_number                = "+1.23456789"
    address_line_1              = "123 Main Street"
    city                        = "Paris"
    zip                         = "75001"
    country                     = "FR"
    vat_identification_code     = "FR12345678901"
    company_identification_code = "123456789"
  }
}
```

### Update Domain Settings

You can update the resource to enable auto-renewal and DNSSEC:

```terraform
resource "scaleway_domain_registration" "test" {
  domain_names      = ["example.com"]
  duration_in_years = 1

  owner_contact {
    legal_form                  = "individual"
    firstname                   = "John"
    lastname                    = "DOE"
    email                       = "john.doe@example.com"
    phone_number                = "+1.23456789"
    address_line_1              = "123 Main Street"
    city                        = "Paris"
    zip                         = "75001"
    country                     = "FR"
    vat_identification_code     = "FR12345678901"
    company_identification_code = "123456789"
  }

  auto_renew = true
  dnssec     = true
}
```

### Purchase Multiple Domains

The following example registers several domains in one go:

```terraform
resource "scaleway_domain_registration" "multi" {
  domain_names      = ["domain1.com", "domain2.com", "domain3.com"]
  duration_in_years = 1

  owner_contact {
    legal_form                  = "individual"
    firstname                   = "John"
    lastname                    = "DOE"
    email                       = "john.doe@example.com"
    phone_number                = "+1.23456789"
    address_line_1              = "123 Main Street"
    city                        = "Paris"
    zip                         = "75001"
    country                     = "FR"
    vat_identification_code     = "FR12345678901"
    company_identification_code = "123456789"
  }
}
```

## Argument Reference

The following arguments are supported:

- `domain_names` (Required, List of String): A list of domain names to be registered.
- `duration_in_years` (Optional, Integer, Default: 1): The registration period in years.
- `project_id` (Optional, String): The Scaleway project ID.
- `owner_contact_id` (Optional, String): The ID of an existing owner contact.
- `owner_contact` (Optional, List): Details of the owner contact.
- `administrative_contact` (Computed, List): Administrative contact information.
- `technical_contact` (Computed, List): Technical contact information.
- `auto_renew` (Optional, Boolean, Default: false): Enables or disables auto-renewal.
- `dnssec` (Optional, Boolean, Default: false): Enables or disables DNSSEC.

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

- `id`: The ID of the domain registration.
- `updated_at`: Timestamp of the last update.
- `expired_at`: Expiration date/time of the domain registration.
- `epp_code`: EPP code(s) associated with the domain.
- `registrar`: Name of the registrar.
- `status`: Status of the domain registration.
- `dns_zones`: List of DNS zones associated with the domain.
- `ds_record`: DNSSEC DS record configuration.
- `task_id`: ID of the task that created the domain.



## Contact Blocks

Each contact block supports the following attributes:

- `legal_form` (Required, String): Legal form of the contact.
- `firstname` (Required, String): First name.
- `lastname` (Required, String): Last name.
- `company_name` (Optional, String): Company name.
- `email` (Required, String): Primary email.
- `phone_number` (Required, String): Primary phone number.
- `address_line_1` (Required, String): Primary address.
- `zip` (Required, String): Postal code.
- `city` (Required, String): City.
- `country` (Required, String): Country code (ISO format).
- `vat_identification_code` (Required, String): VAT identification code.
- `company_identification_code` (Required, String): Company identification code.

## Import

To import an existing domain registration, use:

```bash
terraform import scaleway_domain_registration.test <project_id>/<task_id>
```



