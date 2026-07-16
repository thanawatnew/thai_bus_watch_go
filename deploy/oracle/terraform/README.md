# One-click-ish Oracle Resource Manager stack

This Terraform stack creates the complete network and Always Free-sized VM, then
cloud-init clones and starts Thai Bus Watch automatically. It avoids the Compute
instance creation form that requires an existing VCN/subnet.

## Upload and run

1. In OCI Console open **Developer Services → Resource Manager → Stacks**.
2. Select **Create stack → My configuration → .Zip file**.
3. Upload `thai-bus-watch-oracle-stack.zip` from this directory.
4. Use the root compartment and a current Terraform version, then select **Next**.
5. Enter the tenancy OCID and select the root compartment/home region.
6. For SSH public key, paste the contents of `thai_bus_watch_oracle.pub`. Never
   paste or upload the private key.
7. Prefer your public IPv4 plus `/32` for SSH source CIDR. `0.0.0.0/0` works for
   initial setup but permits SSH attempts from the whole internet.
8. Run **Plan**, review that the VM is `VM.Standard.A1.Flex` with 1 OCPU, 6 GB
   RAM, and a 50 GB boot volume, then run **Apply**.
9. Open the `test_url` output after 5–10 minutes. Cloud-init needs time to install
   Docker and compile the application.

The initial endpoint is HTTP on the public IP. It is useful for testing, but
iPhone geolocation requires HTTPS. After verifying the VM, use the parent
directory's Caddy/domain deployment to enable HTTPS.

If the Apply job reports `Out of host capacity`, change
`availability_domain_index` to another valid zero-based index or retry later.
