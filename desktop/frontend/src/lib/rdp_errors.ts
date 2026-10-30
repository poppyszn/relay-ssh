/**
 * Formats low-level RDP error strings and network failures into clear, plain-language
 * messages explaining what happened and actionable advice on what to do next.
 */
export function formatRdpError(err: string): { title: string; hint: string } {
  const e = (err || "").toLowerCase();
  if (e.includes("open ") && e.includes(" first")) {
    return {
      title: "Jump host requires authentication",
      hint: `${err}. Open a connection to your jump host to sign in before connecting to this RDP host.`,
    };
  }
  if (e.includes("jump") && (e.includes("unreachable") || e.includes("could not connect to jump") || e.includes("jump tunnel failed"))) {
    return {
      title: "Jump host unreachable",
      hint: "Could not establish the SSH tunnel through the jump host. Check that the jump host is online, reachable, and SSH is running on it.",
    };
  }
  if (e.includes("connection refused") || e.includes("port closed")) {
    return {
      title: "Port closed or unreachable",
      hint: "The host refused the connection on port 3389. Check that Remote Desktop is enabled on the target machine and firewall rules allow incoming RDP traffic.",
    };
  }
  if (e.includes("no route to host") || e.includes("target unreachable") || (e.includes("dial tcp") && e.includes("timeout"))) {
    return {
      title: "Host unreachable",
      hint: "Could not reach the host across the network. Check the hostname or IP address, and your network or VPN connection.",
    };
  }
  if (e.includes("nla") || e.includes("logon") || e.includes("auth") || e.includes("access denied") || e.includes("password")) {
    return {
      title: "Login failed (wrong password or NLA refused)",
      hint: "The remote machine refused the login credentials. Check the username, domain, and password, and verify Network Level Authentication permissions.",
    };
  }
  if (e.includes("certificate") && (e.includes("mismatch") || e.includes("changed") || e.includes("rejected"))) {
    return {
      title: "Host certificate changed or rejected",
      hint: "The TLS certificate presented by the host was rejected or differs from the stored fingerprint. Verify the host identity.",
    };
  }
  if (e.includes("timeout") || e.includes("timed out")) {
    return {
      title: "Connection timed out",
      hint: "The connection attempt timed out while waiting for a response from the host. Check network latency and host availability.",
    };
  }
  if (e.includes("closed") || e.includes("eof") || e.includes("reset by peer")) {
    return {
      title: "Session closed by server",
      hint: "The remote server terminated the RDP session. Another user may have logged in, or the session was disconnected by an administrator.",
    };
  }
  return {
    title: "Could not connect",
    hint: err || "An unexpected error occurred while establishing the RDP session.",
  };
}
