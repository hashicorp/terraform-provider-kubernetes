# 0.2.0 (May 10, 2021)

NOTES:

* Upgraded terraform-plugin-go to v0.3.0. Providers built against versions of terraform-plugin-go prior to v0.3.0 will run into compatibility issues due to breaking changes in terraform-plugin-go.

# 0.1.1 (February 10, 2021)

BUG FIXES:

* Compare schemas in an order-insensitive way when deciding whether two server implementations are returning the same schema. ([#18](https://github.com/hashicorp/terraform-plugin-mux/issues/18))
* Surface the difference between schemas when provider and provider_meta schemas differ. ([#18](https://github.com/hashicorp/terraform-plugin-mux/issues/18))

# 0.1.0 (November 02, 2020)

Initial release.
