import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterLink, Router } from '@angular/router';
import { ApiService, Product } from '../../core/api.service';
import { PageHeaderComponent } from '../../shared/index';

@Component({
  selector: 'app-product-create',
  standalone: true,
  imports: [CommonModule, FormsModule, RouterLink, PageHeaderComponent],
  template: `
    <div class="product-create">
      <app-page-header title="New Product">
        <a routerLink="/products" class="back-link">&larr; Back to Products</a>
      </app-page-header>

      <div class="error-banner" *ngIf="error">
        {{ error }}
      </div>

      <div class="form-card">
        <form (ngSubmit)="onSubmit()" #productForm="ngForm">
          <div class="form-grid">
            <div class="form-group">
              <label for="name">Name *</label>
              <input id="name" type="text" [(ngModel)]="formData.name" name="name" #name="ngModel" required>
              <span class="field-error" *ngIf="name.invalid && name.touched">Name is required</span>
            </div>
            <div class="form-group">
              <label for="price">Price *</label>
              <input id="price" type="number" [(ngModel)]="formData.price" name="price" #price="ngModel" required min="0" pattern="^\\d+(\\.\\d{1,2})?$">
              <span class="field-error" *ngIf="price.invalid && price.touched">Valid price is required</span>
            </div>
            <div class="form-group form-group-full">
              <label for="description">Description</label>
              <textarea id="description" [(ngModel)]="formData.description" name="description" rows="3"></textarea>
            </div>
            <div class="form-group">
              <label for="currency">Currency</label>
              <select id="currency" [(ngModel)]="formData.currency" name="currency">
                <option value="" disabled>Select currency</option>
                <option value="USD">USD</option>
                <option value="EUR">EUR</option>
                <option value="GBP">GBP</option>
              </select>
            </div>
          </div>
          <div class="form-actions">
            <a routerLink="/products" class="btn btn-secondary">Cancel</a>
            <button type="submit" class="btn btn-primary" [disabled]="submitting || productForm.invalid">
              {{ submitting ? 'Creating...' : 'Create Product' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  `,
  styles: [`
    .product-create { padding: var(--od-spacing-xl); }
    .back-link { color: var(--od-accent); text-decoration: none; font-size: 14px; display: inline-block; margin-bottom: 8px; }
    .back-link:hover { text-decoration: underline; }
    .error-banner {
      background: var(--od-bg-danger, #fef2f2);
      border: 1px solid var(--od-border-danger, #fecaca);
      border-radius: var(--od-radius);
      padding: 12px 16px;
      color: var(--od-danger, #991b1b);
      font-size: 14px;
      margin-bottom: 16px;
    }
    .form-card {
      background: var(--od-card-bg);
      border-radius: var(--od-radius);
      padding: 24px;
      box-shadow: var(--od-card-shadow);
      max-width: 640px;
    }
    .form-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
    }
    .form-group-full { grid-column: 1 / -1; }
    .form-group { display: flex; flex-direction: column; }
    label { font-size: 13px; font-weight: 500; color: var(--od-text-primary); margin-bottom: 6px; }
    input, select, textarea {
      padding: 8px 12px;
      border: 1px solid var(--od-border);
      border-radius: var(--od-radius-sm);
      font-size: 14px;
      color: var(--od-text-primary);
      background: var(--od-bg-primary);
      outline: none;
      transition: border-color 0.2s;
      font-family: inherit;
    }
    input:focus, select:focus, textarea:focus { border-color: var(--od-accent); }
    input.ng-invalid.ng-touched, select.ng-invalid.ng-touched, textarea.ng-invalid.ng-touched { border-color: var(--od-danger, #ef4444); }
    .field-error { color: var(--od-danger, #ef4444); font-size: 12px; margin-top: 4px; }
    .form-actions {
      display: flex;
      justify-content: flex-end;
      gap: 12px;
      margin-top: 24px;
      padding-top: 16px;
      border-top: 1px solid var(--od-border);
    }
    .btn {
      padding: 8px 20px;
      border-radius: var(--od-radius-sm);
      font-size: 14px;
      font-weight: 500;
      cursor: pointer;
      text-decoration: none;
      border: none;
      transition: background-color 0.2s;
    }
    .btn-primary { background: var(--od-accent); color: white; }
    .btn-primary:hover { opacity: 0.9; }
    .btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
    .btn-secondary { background: var(--od-bg-secondary); color: var(--od-text-primary); display: inline-flex; align-items: center; }
    .btn-secondary:hover { background: var(--od-bg-tertiary); }
  `]
})
export class ProductCreateComponent {
  formData: Partial<Product> = {
    name: '',
    description: '',
    price: 0,
    currency: 'USD'
  };
  submitting = false;
  error = '';
  private merchantId = '';

  constructor(private api: ApiService, private router: Router) {
    this.merchantId = localStorage.getItem('helix_merchant_id') || '';
  }

  onSubmit(): void {
    if (!this.formData.name || !this.formData.price) return;
    this.submitting = true;
    this.error = '';
    this.api.createProduct(this.merchantId, this.formData).subscribe({
      next: (product) => this.router.navigate(['/products', product.id]),
      error: () => {
        this.error = 'Failed to create product. Please check your input and try again.';
        this.submitting = false;
      }
    });
  }
}
