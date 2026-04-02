import numpy as np


def _check_multi_class(multi_class, solver, n_classes):
    """Resolve 'auto' to 'ovr' or 'multinomial'.

    When multi_class='auto':
      - solver='liblinear' -> 'ovr'
      - n_classes == 2 -> 'ovr'
      - otherwise -> 'multinomial'
    """
    if multi_class == "auto":
        if solver == "liblinear":
            multi_class = "ovr"
        elif n_classes == 2:
            multi_class = "ovr"
        else:
            multi_class = "multinomial"

    if multi_class not in ("ovr", "multinomial"):
        raise ValueError(f"Invalid multi_class: {multi_class}")

    return multi_class


class LogisticRegressionCV:
    """Logistic Regression CV (simplified extraction for demo)."""

    def __init__(
        self,
        Cs=10,
        cv=5,
        penalty="l2",
        solver="lbfgs",
        multi_class="auto",
        refit=True,
        l1_ratios=None,
    ):
        self.Cs = Cs
        self.cv = cv
        self.penalty = penalty
        self.solver = solver
        self.multi_class = multi_class
        self.refit = refit
        self.l1_ratios = l1_ratios

    def fit(self, X, y, sample_weight=None):
        solver = self.solver
        n_classes = len(np.unique(y))
        classes = np.unique(y)

        Cs = np.logspace(-4, 4, self.Cs) if isinstance(self.Cs, int) else self.Cs
        self.Cs_ = Cs

        l1_ratios_ = self.l1_ratios or [None]

        multi_class = _check_multi_class(self.multi_class, solver, n_classes)

        n_folds = self.cv
        folds = list(range(n_folds))

        # Simulate cross-validation results
        n_cs_l1 = len(Cs) * len(l1_ratios_)

        if multi_class == "multinomial":
            coefs_paths = np.random.randn(n_classes, n_folds, n_cs_l1, X.shape[1])
            scores = np.random.rand(1, n_folds, n_cs_l1)
            scores = np.tile(scores, (n_classes, 1, 1))
        else:
            coefs_paths = np.random.randn(n_folds, n_cs_l1, X.shape[1])
            scores = np.random.rand(n_classes, n_folds, n_cs_l1)

        self.C_ = list()
        self.l1_ratio_ = list()
        self.coef_ = np.empty((n_classes, X.shape[1]))
        self.intercept_ = np.zeros(n_classes)

        for index, cls in enumerate(classes):
            if multi_class == "ovr":
                class_scores = scores[index]
                class_coefs = coefs_paths
            else:
                class_scores = scores[0]
                class_coefs = coefs_paths

            if self.refit:
                best_index = class_scores.sum(axis=0).argmax()
                best_index_C = best_index % len(Cs)
                self.C_.append(Cs[best_index_C])
                self.l1_ratio_.append(None)

                if multi_class == "multinomial":
                    coef_init = np.mean(class_coefs[:, :, best_index, :], axis=1)
                else:
                    coef_init = np.mean(class_coefs[:, best_index, :], axis=0)

                w = np.random.randn(X.shape[1])
            else:
                best_indices = np.argmax(class_scores, axis=1)

                if self.multi_class == "ovr":
                    w = np.mean(
                        [class_coefs[i, best_indices[i], :] for i in range(len(folds))],
                        axis=0,
                    )
                else:
                    w = np.mean(
                        [
                            class_coefs[:, i, best_indices[i], :]
                            for i in range(len(folds))
                        ],
                        axis=0,
                    )

                best_indices_C = best_indices % len(Cs)
                self.C_.append(np.mean(Cs[best_indices_C]))
                self.l1_ratio_.append(None)

            self.coef_[index] = w[: X.shape[1]]

        self.C_ = np.asarray(self.C_)
        self.l1_ratio_ = np.asarray(self.l1_ratio_)
        return self
