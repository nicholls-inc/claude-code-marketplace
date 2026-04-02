"""Reproducer for scikit-learn__scikit-learn-14087.

LogisticRegressionCV crashes with IndexError when refit=False and
multi_class='auto' (the default since sklearn 0.21).

Expected: fits successfully and stores cross-validated C_ values.
Actual: IndexError: too many indices for array

The traceback points to line ~2194 in logistic.py — an array indexing
expression. But the ROOT CAUSE is at line ~2170, where self.multi_class
(raw user input 'auto') is used instead of multi_class (resolved to 'ovr').

Extracted from sklearn/linear_model/logistic.py (v0.21.2)

SWE-bench: scikit-learn__scikit-learn-14087
Issue: https://github.com/scikit-learn/scikit-learn/issues/14059
Fix: https://github.com/scikit-learn/scikit-learn/pull/14087

"""

import sys

import numpy as np

sys.path.insert(0, "workspace")
from logistic_cv import LogisticRegressionCV

np.random.seed(42)
n_samples, n_features = 100, 3
X = np.random.randn(n_samples, n_features)
# Binary classification: 'auto' resolves to 'ovr' (not 'multinomial')
# This is when the bug triggers — self.multi_class='auto' != 'ovr'
y = np.random.randint(0, 2, n_samples)

print("Testing LogisticRegressionCV with refit=False, multi_class='auto'...")
print(f"  X.shape={X.shape}, n_classes=2 (binary)")
print()

# This crashes with the bug
try:
    model = LogisticRegressionCV(
        multi_class="auto",  # Default since 0.21 — resolves to 'ovr' for binary
        refit=False,
        solver="lbfgs",
        cv=3,
    )
    model.fit(X, y)
    print("SUCCESS: fit completed without error")
except IndexError as e:
    print(f"CRASH: IndexError: {e}")
    print()
    print("The error traceback points to array indexing (the crash site).")
    print(
        "But the ROOT CAUSE is 25 lines above: self.multi_class should be multi_class."
    )
    print()
    print("  self.multi_class = 'auto'  (raw user input)")
    print("  multi_class      = 'ovr'   (resolved by _check_multi_class)")
    print()
    print("  if self.multi_class == 'ovr':  # FALSE — it's 'auto'")
    print(
        "      w = np.mean([coefs_paths[i, ...]])     # 3D indexing (correct for OVR)"
    )
    print("  else:")
    print("      w = np.mean([coefs_paths[:, i, ...]])  # 4D indexing (WRONG for OVR)")
    print("                                              # -> IndexError")
