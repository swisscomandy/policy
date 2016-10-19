# policy
2nd version of sow implementation
Use the src port instead of dst port.
Modified the iptables SNAT postrouting chain
To run, add the line -policyURL=<ip> \    to the startup command. Note: use private ip in the same stack.
