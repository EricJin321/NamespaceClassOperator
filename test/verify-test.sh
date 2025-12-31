# Verification script to check test results
# Run this after running run-test.sh

echo "🔍 Verifying test results..."
echo "============================"

echo ""
echo "1. Check CRDs:"
kubectl get crd | grep -E "(appconfig|namespaceclass)"

echo ""
echo "2. Check NamespaceClassItem:"
kubectl get namespaceclassitem -n namespaceclassoperator-system

echo ""
echo "3. Check NamespaceClass:"
kubectl get namespaceclass -n namespaceclassoperator-system

echo ""
echo "4. Check test namespace:"
kubectl get namespace test-app-namespace

echo ""
echo "5. Check AppConfig in test namespace:"
kubectl get appconfig -n test-app-namespace

echo ""
echo "6. Check AppConfig content:"
kubectl get appconfig -n test-app-namespace -o yaml

echo ""
echo "✅ Verification complete!"