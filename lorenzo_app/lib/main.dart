import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter_v2ray_client/flutter_v2ray.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const LorenzoVpnApp());
}

class LorenzoVpnApp extends StatelessWidget {
  const LorenzoVpnApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Lorenzo VPN',
      debugShowCheckedModeBanner: false,
      theme: ThemeData.dark().copyWith(
        scaffoldBackgroundColor: const Color(0xFF090D16),
        colorScheme: const ColorScheme.dark(
          primary: Color(0xFF00F2FE),
          secondary: Color(0xFFFF2A55),
          surface: Color(0xFF121826),
        ),
      ),
      home: const HomeScreen(),
    );
  }
}

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen>
    with SingleTickerProviderStateMixin {
  // Built-in VLESS Link configured for Lorenzo VPN
  static const String vlessLink =
      "vless://0e8d9728-b93f-484d-884e-e0309714c61f@lorenzo-vpn-production.up.railway.app:443?path=%2Fvless&security=tls&encryption=none&type=ws#LorenzoVPN";

  late final FlutterV2ray _v2ray;
  V2RayStatus _status = V2RayStatus();
  bool _isConnected = false;
  bool _isConnecting = false;

  Timer? _timer;
  int _secondsElapsed = 0;

  late AnimationController _pulseController;
  late Animation<double> _pulseAnimation;

  @override
  void initState() {
    super.initState();
    _initV2ray();

    _pulseController = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 2),
    )..repeat(reverse: true);

    _pulseAnimation = Tween<double>(begin: 1.0, end: 1.12).animate(
      CurvedAnimation(parent: _pulseController, curve: Curves.easeInOut),
    );
  }

  Future<void> _initV2ray() async {
    _v2ray = FlutterV2ray(
      onStatusChanged: (status) {
        setState(() {
          _status = status;
          if (status.state == 'CONNECTED') {
            _isConnected = true;
            _isConnecting = false;
            _startTimer();
          } else if (status.state == 'DISCONNECTED') {
            _isConnected = false;
            _isConnecting = false;
            _stopTimer();
          } else if (status.state == 'CONNECTING') {
            _isConnecting = true;
          }
        });
      },
    );

    try {
      await _v2ray.initializeV2Ray();
    } catch (_) {}
  }

  void _startTimer() {
    _timer?.cancel();
    _secondsElapsed = 0;
    _timer = Timer.periodic(const Duration(seconds: 1), (_) {
      setState(() {
        _secondsElapsed++;
      });
    });
  }

  void _stopTimer() {
    _timer?.cancel();
    _secondsElapsed = 0;
  }

  String _formatDuration(int seconds) {
    final h = (seconds ~/ 3600).toString().padLeft(2, '0');
    final m = ((seconds % 3600) ~/ 60).toString().padLeft(2, '0');
    final s = (seconds % 60).toString().padLeft(2, '0');
    return "$h:$m:$s";
  }

  Future<void> _toggleConnection() async {
    if (_isConnected) {
      await _v2ray.stopV2Ray();
      setState(() {
        _isConnected = false;
        _isConnecting = false;
        _stopTimer();
      });
    } else {
      setState(() {
        _isConnecting = true;
      });

      try {
        final hasPermission = await _v2ray.requestPermission();
        if (!hasPermission) {
          setState(() {
            _isConnecting = false;
          });
          return;
        }

        final parser = V2ray.parseFromURL(vlessLink);
        await _v2ray.startV2Ray(
          remark: "Lorenzo VPN (Amsterdam)",
          config: parser.getFullConfiguration(),
          proxyOnly: false,
        );
      } catch (e) {
        setState(() {
          _isConnecting = false;
          _isConnected = false;
        });
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text("فشل الاتصال: $e"),
              backgroundColor: Colors.redAccent,
            ),
          );
        }
      }
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    _pulseController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: RadialGradient(
            center: Alignment(0, -0.4),
            radius: 1.2,
            colors: [
              Color(0xFF131C2E),
              Color(0xFF090D16),
            ],
          ),
        ),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
            child: Column(
              children: [
                // Top App Bar / Brand Header
                _buildHeader(),
                const Spacer(flex: 1),

                // Server Details Card
                _buildServerCard(),
                const Spacer(flex: 2),

                // Central Power Button with Glow
                _buildPowerButton(),
                const SizedBox(height: 24),

                // Status Text
                _buildStatusText(),
                const Spacer(flex: 2),

                // Live Stats (Timer, Speed)
                _buildStatsCard(),
                const Spacer(flex: 1),

                // Motto
                const Text(
                  "SAME BOY... BIGGER VISION 👑",
                  style: TextStyle(
                    color: Color(0xFF64748B),
                    fontSize: 12,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 2,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildHeader() {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Container(
          width: 44,
          height: 44,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            border: Border.all(color: const Color(0xFF00F2FE), width: 1.5),
            boxShadow: [
              BoxShadow(
                color: const Color(0xFF00F2FE).withOpacity(0.3),
                blurRadius: 10,
              ),
            ],
          ),
          child: ClipOval(
            child: Image.asset(
              'assets/lorenzo.jpg',
              fit: BoxFit.cover,
            ),
          ),
        ),
        const SizedBox(width: 12),
        Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: const [
            Text(
              "LORENZO VPN",
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w900,
                letterSpacing: 1.5,
                color: Colors.white,
              ),
            ),
            Text(
              "THE STRICT LEADER • VLESS PROXY",
              style: TextStyle(
                fontSize: 10,
                fontWeight: FontWeight.w600,
                color: Color(0xFF94A3B8),
                letterSpacing: 0.8,
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildServerCard() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
      decoration: BoxDecoration(
        color: const Color(0xFF141A28),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: const Color(0xFF222D42)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withOpacity(0.3),
            blurRadius: 15,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: const Color(0xFF1E283D),
              borderRadius: BorderRadius.circular(12),
            ),
            child: const Text("🇳🇱", style: TextStyle(fontSize: 24)),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: const [
                Text(
                  "Netherlands (Amsterdam)",
                  style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w700,
                    color: Colors.white,
                  ),
                ),
                SizedBox(height: 2),
                Text(
                  "Fast Gaming & Roblox Bypass",
                  style: TextStyle(
                    fontSize: 12,
                    color: Color(0xFF94A3B8),
                  ),
                ),
              ],
            ),
          ),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              color: const Color(0xFF10B981).withOpacity(0.15),
              borderRadius: BorderRadius.circular(20),
              border: Border.all(
                color: const Color(0xFF10B981).withOpacity(0.4),
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: const [
                Icon(Icons.bolt, size: 14, color: Color(0xFF10B981)),
                SizedBox(width: 2),
                Text(
                  "102 ms",
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w800,
                    color: Color(0xFF10B981),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPowerButton() {
    final bool active = _isConnected;
    final Color glowColor = active
        ? const Color(0xFF10B981)
        : (_isConnecting
            ? const Color(0xFFF59E0B)
            : const Color(0xFF00F2FE));

    return GestureDetector(
      onTap: _isConnecting ? null : _toggleConnection,
      child: ScaleTransition(
        scale: active ? _pulseAnimation : const AlwaysStoppedAnimation(1.0),
        child: Container(
          width: 180,
          height: 180,
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: const Color(0xFF121927),
            border: Border.all(color: glowColor, width: 3),
            boxShadow: [
              BoxShadow(
                color: glowColor.withOpacity(active ? 0.45 : 0.25),
                blurRadius: active ? 40 : 25,
                spreadRadius: active ? 8 : 2,
              ),
            ],
          ),
          child: Center(
            child: _isConnecting
                ? SizedBox(
                    width: 50,
                    height: 50,
                    child: CircularProgressIndicator(
                      color: glowColor,
                      strokeWidth: 3.5,
                    ),
                  )
                : Icon(
                    Icons.power_settings_new_rounded,
                    size: 74,
                    color: glowColor,
                  ),
          ),
        ),
      ),
    );
  }

  Widget _buildStatusText() {
    String text = "اضغط للاتصال";
    Color color = const Color(0xFF94A3B8);

    if (_isConnected) {
      text = "محمي بالكامل ومشفّر 🛡️";
      color = const Color(0xFF10B981);
    } else if (_isConnecting) {
      text = "جاري الاتصال بالسيرفر...";
      color = const Color(0xFFF59E0B);
    }

    return Text(
      text,
      style: TextStyle(
        fontSize: 16,
        fontWeight: FontWeight.w700,
        color: color,
      ),
    );
  }

  Widget _buildStatsCard() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
      decoration: BoxDecoration(
        color: const Color(0xFF141A28),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: const Color(0xFF222D42)),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          _buildStatItem(
            title: "وقت الاتصال",
            value: _formatDuration(_secondsElapsed),
            icon: Icons.timer_outlined,
            color: const Color(0xFF00F2FE),
          ),
          Container(width: 1, height: 35, color: const Color(0xFF222D42)),
          _buildStatItem(
            title: "التحميل (Download)",
            value: _status.downloadSpeed.isNotEmpty
                ? _status.downloadSpeed
                : "0 KB/s",
            icon: Icons.arrow_downward_rounded,
            color: const Color(0xFF10B981),
          ),
          Container(width: 1, height: 35, color: const Color(0xFF222D42)),
          _buildStatItem(
            title: "الرفع (Upload)",
            value: _status.uploadSpeed.isNotEmpty
                ? _status.uploadSpeed
                : "0 KB/s",
            icon: Icons.arrow_upward_rounded,
            color: const Color(0xFFFF2A55),
          ),
        ],
      ),
    );
  }

  Widget _buildStatItem({
    required String title,
    required String value,
    required IconData icon,
    required Color color,
  }) {
    return Column(
      children: [
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 14, color: color),
            const SizedBox(width: 4),
            Text(
              title,
              style: const TextStyle(
                fontSize: 11,
                color: Color(0xFF94A3B8),
              ),
            ),
          ],
        ),
        const SizedBox(height: 6),
        Text(
          value,
          style: const TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w800,
            color: Colors.white,
          ),
        ),
      ],
    );
  }
}
