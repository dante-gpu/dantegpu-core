import { useWallet } from "@solana/wallet-adapter-react";
import { User, Mail, Wallet as WalletIcon, Globe, LogOut, Sun, Moon } from "lucide-react";
import { useAuth } from "@/providers/AuthProvider";
import { useTheme } from "@/providers/ThemeProvider";
import { Card, CardBody, CardHeader } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { WalletButton } from "@/components/WalletButton";
import { solanaCluster } from "@/lib/solana";
import { shortAddress } from "@/lib/format";
import { cn } from "@/lib/cn";

export default function Settings() {
  const { user, logout } = useAuth();
  const { publicKey } = useWallet();
  const { theme, setTheme } = useTheme();
  const cluster = solanaCluster();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-ink-50">Settings</h1>
        <p className="mt-1 text-sm text-ink-400">Manage your account and connections.</p>
      </div>

      <Card>
        <CardHeader title="Profile" />
        <CardBody className="space-y-4 pt-1">
          <Field icon={<User className="size-4" />} label="Username" value={user?.username ?? "-"} />
          <Field icon={<Mail className="size-4" />} label="Email" value={user?.email ?? "Not set"} />
          {user?.role && (
            <Field icon={<User className="size-4" />} label="Role" value={user.role} />
          )}
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Appearance" description="Choose how the console looks." />
        <CardBody className="pt-1">
          <div className="inline-flex rounded-lg border border-ink-600 bg-ink-850 p-1">
            {([
              { key: "dark" as const, label: "Dark", icon: Moon },
              { key: "light" as const, label: "Light", icon: Sun },
            ]).map(({ key, label, icon: Icon }) => (
              <button
                key={key}
                onClick={() => setTheme(key)}
                className={cn(
                  "flex items-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors",
                  theme === key ? "bg-ember-500 text-white" : "text-ink-300 hover:text-ink-50",
                )}
              >
                <Icon className="size-4" />
                {label}
              </button>
            ))}
          </div>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Solana connection" description="Where balances and settlements are read from." />
        <CardBody className="space-y-4 pt-1">
          <div className="flex items-center justify-between">
            <Field icon={<Globe className="size-4" />} label="Cluster" value={cluster} inline />
            <Badge tone={cluster === "mainnet-beta" ? "positive" : "caution"}>
              {cluster === "mainnet-beta" ? "Mainnet" : "Test cluster"}
            </Badge>
          </div>
          <div className="flex items-center justify-between gap-4">
            <Field
              icon={<WalletIcon className="size-4" />}
              label="Connected wallet"
              value={publicKey ? shortAddress(publicKey.toBase58(), 6, 6) : "Not connected"}
              inline
            />
            <WalletButton size="sm" />
          </div>
        </CardBody>
      </Card>

      <Card>
        <CardHeader title="Session" />
        <CardBody className="pt-1">
          <Button variant="danger" onClick={logout}>
            <LogOut className="size-4" /> Sign out
          </Button>
        </CardBody>
      </Card>
    </div>
  );
}

function Field({
  icon,
  label,
  value,
  inline,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  inline?: boolean;
}) {
  return (
    <div className={inline ? "" : "flex items-center justify-between"}>
      <span className="flex items-center gap-2 text-sm text-ink-300">
        <span className="text-ink-500">{icon}</span>
        {label}
      </span>
      <span className={inline ? "mt-0.5 block text-sm font-medium text-ink-50" : "text-sm font-medium text-ink-50"}>
        {value}
      </span>
    </div>
  );
}
